package eventstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/manucorporat/sse"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rwynn/gtm"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MongoesConfig struct {
	MongoUri string `json:"mongouri"`
}

type mongoesEvent struct {
	Event      *Event
	SequenceID uint64
}

type mongoes struct {
	eventInbound     EventStream
	registered       map[Client]bool
	register         chan Registration
	unregister       chan Client
	session          *mgo.Session
	lastPublishedSeq uint64
	ctx              *gtm.OpCtx
	cfg              *MongoesConfig
}

func (es *mongoes) OneEvent(parentSpan opentracing.Span, eventID string) (*mongoesEvent, error) {

	span := opentracing.StartSpan("one_event", opentracing.ChildOf(parentSpan.Context()))
	defer span.Finish()
	span.SetTag("param.eventId", eventID)
	span.SetTag("call", "external_db_service_call")

	var url *url.URL
	url, err := url.Parse("http://einbuchen-crest:3500/einbuchen/events")
	if err != nil {
		return nil, err
	}

	q := url.Query()
	q.Add(`query`, `{"event.id":"`+eventID+`"}`)
	url.RawQuery = q.Encode()

	httpClient := &http.Client{}
	httpReq, _ := http.NewRequest("GET", url.String(), nil)

	opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(httpReq.Header))

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var e []mongoesEvent
	json.Unmarshal(body, &e)
	if len(e) != 1 {
		return nil, err
	}

	return &e[0], nil
}

func (es *mongoes) replay(lastEventID string, to Client) {
	var replayStartSeq uint64
	col := es.session.DB("").C("events")
	if len(lastEventID) == 0 {
		replayStartSeq = 0
	} else {
		var e mongoesEvent
		err := col.Find(bson.M{"event.id": lastEventID}).One(&e)
		if err == nil {
			replayStartSeq = e.SequenceID
		}
	}
	afterStart := bson.M{"sequenceid": bson.M{"$gt": replayStartSeq}}
	includesEnd := bson.M{"sequenceid": bson.M{"$lte": es.lastPublishedSeq}}
	seqRange := bson.M{"$and": []bson.M{afterStart, includesEnd}}
	iter := col.Find(seqRange).Sort("sequenceid").Iter()

	var result mongoesEvent
	for iter.Next(&result) {
		to <- result.Event
	}
	iter.Close()
}

func (es *mongoes) nextSequenceId() uint64 {
	var result struct {
		Counter uint64
	}
	col := es.session.DB("").C("sequences")
	col.FindId("events").Apply(mgo.Change{
		Update:    bson.M{"$inc": bson.M{"counter": 1}},
		ReturnNew: true,
	}, &result)
	return result.Counter
}

func (es *mongoes) lastSequnceId() uint64 {
	var result struct {
		Counter uint64
	}
	es.session.DB("").C("sequences").FindId("events").One(&result)
	return result.Counter
}

func (es *mongoes) Start() error {
	if sess, err := mgo.Dial(es.cfg.MongoUri); err != nil {
		return fmt.Errorf("cannot connect to mongodb, uri: %s - %s", es.cfg.MongoUri, err.Error())
	} else {
		es.session = sess
	}

	es.session.SetMode(mgo.Monotonic, true)
	eventsCol := es.session.DB("").C("events")
	eventsCol.EnsureIndexKey("event.id")
	eventsCol.EnsureIndexKey("sequenceid")
	// only insert if not exists
	seqCol := es.session.DB("").C("sequences")
	seqCol.Insert(bson.M{"_id": "events", "counter": uint64(0)})
	es.lastPublishedSeq = es.lastSequnceId()

	seqNS := es.session.DB("").C("events").FullName
	es.ctx = gtm.Start(es.session, &gtm.Options{
		Filter: func(op *gtm.Op) bool {
			return op.Namespace == seqNS && op.IsInsert()
		},
	})

	sessionCheckTimer := time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case reg := <-es.register:
				es.registered[reg.client] = true
				es.replay(reg.lastEventID, reg.client)
				log.Printf("new client registered, replaying events")

			case awayClient := <-es.unregister:
				delete(es.registered, awayClient)
				close(awayClient)
				log.Printf("client unregistered, left %d clients", len(es.registered))

			case e := <-es.eventInbound:
				es.session.DB("").C("events").Insert(mongoesEvent{
					Event:      e,
					SequenceID: es.nextSequenceId(),
				})

			case op := <-es.ctx.OpC:
				var e mongoesEvent
				if bytes, err := bson.Marshal(op.Data); err != nil {
					log.Printf("cannot serialize event: %s\n", err)
				} else {
					if err := bson.Unmarshal(bytes, &e); err != nil {
						log.Printf("cannot unserialize event: %s\n", err)
					}
				}

				es.lastPublishedSeq = e.SequenceID
				for c := range es.registered {
					c <- e.Event
				}

			case <-sessionCheckTimer.C:
				if err := es.session.Ping(); err != nil {
					log.Panic("mongo session lost")
				}
			}
		}
	}()
	return nil
}

func (es *mongoes) Stop() {
	if es.session != nil {
		es.session.Close()
	}
	if es.ctx != nil {
		es.ctx.Stop()
	}
}

func NewMongoes(conf *MongoesConfig) EventStore {
	return &mongoes{
		registered:   make(map[Client]bool),
		eventInbound: make(EventStream),
		register:     make(chan Registration),
		unregister:   make(chan Client),
		cfg:          conf,
	}
}

func (b *mongoes) EventInbound() EventInbound {
	return b.eventInbound
}

func (b *mongoes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	messageChan := make(Client)
	b.register <- Registration{
		client:      messageChan,
		lastEventID: r.Header.Get("last-event-id"),
	}

	// Listen to the closing of the http connection via the CloseNotifier
	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		b.unregister <- messageChan
		log.Println("HTTP connection just closed.")
	}()

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", sse.ContentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// keep alive
	keepalive := time.NewTicker(time.Second * 15)
	for {
		select {
		case <-keepalive.C:
			w.Write([]byte(": ping\n"))
			f.Flush()

		// Read from our messageChan.
		case msg, open := <-messageChan:
			if !open {
				goto End
			}

			// Write to the ResponseWriter, `w`.
			buff := bytes.NewBuffer(make([]byte, 0, 200))
			sse.Encode(buff, sse.Event{
				Id:    msg.Id,
				Event: msg.Type,
				Data:  msg.Payload,
			})
			buff.WriteTo(w)
			f.Flush()
		}
	}
End:
	// Done.
	log.Println("Finished HTTP request at ", r.URL.Path)
	keepalive.Stop()
}

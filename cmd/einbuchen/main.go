package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	thislog "github.com/container-bootcamp/einbuchen/pkg/log"
	"github.com/container-bootcamp/einbuchen/pkg/tracing"
	"github.com/gorilla/mux"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/satori/go.uuid"
	"github.com/uber/jaeger-lib/metrics/go-kit"
	"github.com/uber/jaeger-lib/metrics/go-kit/expvar"
	"go.uber.org/zap"

	"flag"

	"github.com/container-bootcamp/einbuchen/cmd/einbuchen/eventstore"
)

type Config struct {
	Mongoes       eventstore.MongoesConfig `json:"mongoes"`
	BindInterface string                   `json:"bind-interface"`
	BindPort      string                   `json:"bind-port"`
}

type Buch struct {
	Isbn             string
	Titel            string
	Autor            string
	KurzBeschreibung string
}

func main() {
	conf := Config{
		Mongoes: eventstore.MongoesConfig{
			MongoUri: "mongodb://localhost:27017/einbuchen",
		},
		BindInterface: "0.0.0.0",
		BindPort:      "8080",
	}
	var confFileName = flag.String("conf", "none uses defaultconfig instead", "config file")
	flag.Parse()

	if file, err := os.Open(*confFileName); err == nil {
		if err2 := json.NewDecoder(file).Decode(&conf); err2 != nil {
			log.Printf("cannot read: config file %s: %s\n", *confFileName, err2.Error())
			log.Println("use default config")
		} else {
			log.Printf("using conf file %s\n", *confFileName)
		}
	} else {
		log.Println("use default config")
	}

	logger, _ := zap.NewDevelopment()
	logfac := thislog.NewFactory(logger.With(zap.String("service", "einbuchen")))
	metricsFactory := xkit.Wrap("", expvar.NewFactory(10))
	opentracing.InitGlobalTracer(tracing.Init("einbuchen", metricsFactory.Namespace("frontend", nil), logfac))

	es := eventstore.NewMongoes(&conf.Mongoes)

	if err := es.Start(); err != nil {
		log.Fatalf("Exiting: %s\n", err.Error())
	}
	defer es.Stop()

	sendBuchEingebuchtEvent := func(book *Buch) error {
		buff := bytes.NewBuffer(make([]byte, 0, 400))
		e := json.NewEncoder(buff)
		e.Encode(book)
		es.EventInbound() <- &eventstore.Event{
			Type:    "BuchEingebucht",
			Id:      uuid.NewV4().String(),
			Payload: string(buff.Bytes()),
		}
		return nil
	}

	getBook := func(parentSpan opentracing.Span, eventId string) (*Buch, error) {

		span := opentracing.StartSpan("get_book", opentracing.ChildOf(parentSpan.Context()))
		defer span.Finish()
		span.SetTag("param.eventId", eventId)

		e, err := es.OneEvent(span, eventId)
		if err != nil {
			log.Printf("Event Id %s not found\n", eventId)
			return nil, err
		}
		var b Buch
		err = json.Unmarshal([]byte(e.Event.Payload), &b)
		return &b, err
	}

	router := mux.NewRouter()
	router.HandleFunc("/books", BooksHandlerfunc(sendBuchEingebuchtEvent)).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/book/{eventid}", OneBookHandlerfunc(getBook)).Methods(http.MethodGet)
	router.Handle("/events", es).Methods(http.MethodGet)
	router.HandleFunc("/sse-test", SsetestHandlerfunc).Methods(http.MethodGet)
	router.PathPrefix("/assets/").Handler(http.FileServer(http.Dir("resources/"))).Methods(http.MethodGet)

	log.Printf("listen to %s:%s\n", conf.BindInterface, conf.BindPort)
	if err := http.ListenAndServe(fmt.Sprintf("%s:%s", conf.BindInterface, conf.BindPort), router); err != nil {
		log.Fatal(err)
	}
}

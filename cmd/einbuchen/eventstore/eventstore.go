package eventstore

import (
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
)

type Event struct {
	Id      string
	Type    string
	Payload string
}

type EventStream = chan *Event
type EventInbound = chan<- *Event

type Registration struct {
	client      Client
	lastEventID string
}

type Client = chan *Event

type EventStore interface {
	http.Handler
	Start() error
	Stop()
	EventInbound() EventInbound
	OneEvent(opentracing.Span, string) (*mongoesEvent, error)
}

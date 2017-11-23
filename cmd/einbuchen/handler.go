package main

import (
	"net/http"

	"github.com/gorilla/mux"
	opentracing "github.com/opentracing/opentracing-go"
	"gitlab.innoq.com/container-bootcamp-demo/einbuchen/cmd/einbuchen/view"
)

var templates = view.HtmlTmpl()

func BooksHandlerfunc(sendBuchEingebuchtEvent func(*Buch) error) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {

		switch req.Method {
		case http.MethodPost:
			req.ParseForm()
			book := &Buch{
				Isbn:             req.FormValue("isbn"),
				Titel:            req.FormValue("title"),
				Autor:            req.FormValue("author"),
				KurzBeschreibung: req.FormValue("desc_short"),
			}
			// TODO validate
			sendBuchEingebuchtEvent(book)
			templates.ExecuteTemplate(res, "einbuchen-form", book)

		case http.MethodGet:
			book := &Buch{}
			templates.ExecuteTemplate(res, "einbuchen-form", book)
		}
	}
}

func OneBookHandlerfunc(getBook func(opentracing.Span, string) (*Buch, error)) func(res http.ResponseWriter, req *http.Request) {

	return func(res http.ResponseWriter, req *http.Request) {

		span := opentracing.StartSpan("one_book_handler")
		span.SetTag("param.eventid", mux.Vars(req)["eventid"])
		defer span.Finish()

		switch req.Method {
		case http.MethodGet:
			vars := mux.Vars(req)
			eventid := vars["eventid"]
			if book, err := getBook(span, eventid); err == nil {
				templates.ExecuteTemplate(res, "einbuchen-get", book)
			} else {
				http.NotFound(res, req)
			}
		}
	}
}

func SsetestHandlerfunc(res http.ResponseWriter, req *http.Request) {
	templates.ExecuteTemplate(res, "sse-test", nil)
}

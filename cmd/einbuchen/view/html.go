package view

import "html/template"

const einbuchenFormTmpl = `
{{ define "einbuchen-form" }}
<!DOCTYPE html>
<html>
<head>
    <title>Einbuchen</title>
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0-beta/css/bootstrap.min.css" integrity="sha384-/Y6pD6FV/Vv2HJnA6t+vslU6fwYXjCFtcEpHbNJ0lyAFsXTsjBbfaDjzALeQsN6M" crossorigin="anonymous">
    <link rel="stylesheet" href="/css/bibliothek.css">
    <link rel="stylesheet" href="assets/dist/main.css"/>
</head>
<body>
<esi:include src="http://assets/header.html" />
<main>
    <div class="o-box">
        <h1>Einbuchen</h1>
        <k8s-form data-target="main">
            <form method="POST">
                <div class="input-group">
                    <input class="form-control" type="text" placeholder="ISBN" name="isbn" id="isbn" value="{{.Isbn}}">
                </div>
                <br />

                <div class="input-group">
                    <input class="form-control" type="text" placeholder="Buchtitel" name="title" id="title" value="{{.Titel}}">
                </div>
                <br />

                <div class="input-group">
                    <input class="form-control" type="text" placeholder="Autor" name="author" id="author" value="{{.Autor}}">
                </div>
                <br />

                <div class="input-group">
                    <textarea class="form-control" placeholder="Kurzbeschreibung" name="desc_short" id="desc_short">{{.KurzBeschreibung}}</textarea>
                </div>
                <br />

                <button type="submit" class="btn btn-dark">Einbuchen</button>
            </form>
        </k8s-form>
    </div>
</main>
<esi:include src="http://assets/footer.html" />
</body>
</html>
{{ end }}

{{ define "einbuchen-get" }}
<!DOCTYPE html>
<html>
<head>
    <title>Buch Detailseite</title>
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0-beta/css/bootstrap.min.css" integrity="sha384-/Y6pD6FV/Vv2HJnA6t+vslU6fwYXjCFtcEpHbNJ0lyAFsXTsjBbfaDjzALeQsN6M" crossorigin="anonymous">
    <link rel="stylesheet" href="/css/bibliothek.css">
    <link rel="stylesheet" href="assets/dist/main.css"/>
</head>
<body>
<esi:include src="http://assets/header.html" />
<main>
    <div class="card bg-light mb-3">
        <div class="card-body">
            <h4 class="card-title">{{.Titel}}</h4>
            <h6 class="card-subtitle mb-2 text-muted">{{.Autor}}</h6>
            <p class="card-text">{{.KurzBeschreibung}}</p>
            <div>
                <iq-embedded-link data-target="main">
                    <a href="/ausleihen/book/{{.Isbn}}/lent">Status Verfügbarkeit</a>
                </iq-embedded-link>
            </div>
            ISBN: <span class="badge badge-info">{{.Isbn}}</span>
            <a href="/ausleihen/book/{{.Isbn}}" class="btn btn-primary">Ausleihen</a>
        </div>
    </div>
</main>
<esi:include src="http://assets/footer.html" />
</body>
</html>
{{ end }}

{{ define "sse-test" }}
<!DOCTYPE html>
<html>
<body>

<h1>Getting server updates</h1>
<div id="result"></div>

<script>
    let source = new EventSource("events");
    source.addEventListener('BuchEingebucht', function(event) {
        document.getElementById("result").innerHTML += event.data + "<br>";
    }, false);
</script>
</body>
</html>
{{ end }}
`

func HtmlTmpl() *template.Template {
	return template.Must(template.New("main").Parse(einbuchenFormTmpl))
}

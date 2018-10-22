package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/apex/log"
	jsonl "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	"github.com/tj/go/http/response"
)

func init() {
	if os.Getenv("UP_STAGE") == "" {
		log.SetHandler(text.Default)
	} else {
		log.SetHandler(jsonl.Default)
	}
}

type viewCount int32

var v viewCount

func (n *viewCount) inc() (currentcount int32) {
	return atomic.AddInt32((*int32)(n), 1)
}

func inc(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, v.inc())
}

func main() {
	flag.Parse()

	http.HandleFunc("/favicon.ico", http.NotFound)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", countpage)

	http.HandleFunc("/inc/", inc)
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatalf("error listening: %s", err)
	}

}

func countpage(w http.ResponseWriter, r *http.Request) {

	t := template.Must(template.New("index").ParseFiles("static/index.tmpl"))

	envmap := make(map[string]string)
	for _, e := range os.Environ() {
		ep := strings.SplitN(e, "=", 2)
		// Skip potentially security sensitive AWS stuff
		if ep[0] == "AWS_SECRET_ACCESS_KEY" {
			continue
		}
		if ep[0] == "AWS_SESSION_TOKEN" {
			continue
		}

		envmap[ep[0]] = ep[1]
	}

	// https://golang.org/pkg/net/http/#Request
	envmap["METHOD"] = r.Method
	envmap["PROTO"] = r.Proto
	envmap["CONTENTLENGTH"] = fmt.Sprintf("%d", r.ContentLength)
	envmap["TRANSFERENCODING"] = strings.Join(r.TransferEncoding, ",")
	envmap["REMOTEADDR"] = r.RemoteAddr
	envmap["HOST"] = r.Host
	envmap["REQUESTURI"] = r.RequestURI

	err := t.ExecuteTemplate(w, "index.tmpl", struct {
		Count  int32
		Env    map[string]string
		Header http.Header
	}{v.inc(), envmap, r.Header})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

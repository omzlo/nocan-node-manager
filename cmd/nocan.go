package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"pannetrat.com/nocan"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/intelhex"
	"strings"
)

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I hate %s!", r.URL.Path[1:])
}

type CheckRouter struct {
	handler http.Handler
}

func (cr *CheckRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clog.Debug("%s request from %s to %s", r.Method, r.RemoteAddr, r.RequestURI)
	if len(r.URL.Path) > 1 {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
	}
	cr.handler.ServeHTTP(w, r)
}

func main() {
	var id [8]byte

	fmt.Println("Start")

	//if se != nil {
	//	se.Close()
	//}

	data, _ := ioutil.ReadFile("test.hex")

	ih := intelhex.New()
	err := ih.Load(strings.NewReader(string(data)))
	if err != nil {
		clog.Error("%s", err.Error())
	}

	core := nocan.NewCoreEndpoint()
	core.Topics.Model.Register("foo")
	core.Topics.Model.Register("pizza")
	nocan.StringToUid("01:02:03:04:05:06:07:88", id[:])
	core.Nodes.Model.Register(id)

	ports := nocan.NewPortModel()
	ports.Add(&nocan.LogEndpoint{})
	ports.Add(core)
	se := nocan.NewSerialEndpoint("/dev/cu.usbmodem12341")
	if se != nil {
		ports.Add(se)
	}

	router := httprouter.New()
	router.GET("/topic", core.Topics.Index)
	router.GET("/topic/*topic", core.Topics.Show)
	router.PUT("/topic/*topic", core.Topics.Update)
	router.GET("/nodes", core.Nodes.Index)
	router.GET("/nodes/:node", core.Nodes.Show)
	//http.Handle("/api/topic/", &TopicHandler{Topics: tm})
	//http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/Users/apannetrat/go/src/pannetrat.com/nocan/static/"))))
	//http.HandleFunc("/", defaultHandler)
	go ports.ListenAndServe()
	http.ListenAndServe(":8888", &CheckRouter{router})
	fmt.Println("Done")
}

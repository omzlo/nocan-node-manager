package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"pannetrat.com/nocan"
	"pannetrat.com/nocan/intelhex"
	"pannetrat.com/nocan/log"
	"strings"
)

/*
func (th *TopicHandler) Toto(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	topicName := r.URL.Path[len("/topic/"):]

	if len(topicName) == 0 {
		if r.Method == "GET" {
			for k, _ := range th.Topics.Names {
				fmt.Fprintf(w, "%s\n", k)
			}
			return
		}
		http.Error(w, "Operation not permitted", http.StatusMethodNotAllowed)
		return
	}

	topic := th.Topics.FindByName(topicName)

	switch r.Method {
	case "GET":
		if topic >= 0 {
			content, _ := th.Topics.GetContent(topic)
			w.Write(content)
			return
		}
		http.Error(w, "Topic does not exist.", http.StatusNotFound)
		return
	case "PUT":
		if topic >= 0 {
			body, _ := ioutil.ReadAll(r.Body)
			th.Topics.SetContent(topic, body)
			return
		}
		http.Error(w, "Topic does not exist.", http.StatusNotFound)
		return
	default:
		http.Error(w, "Only GET and PUT are allowed.", http.StatusMethodNotAllowed)
	}
	return
}

type NodeHandler struct {

}
*/
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I hate %s!", r.URL.Path[1:])
}

type CheckRouter struct {
	handler http.Handler
}

func (cr *CheckRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Log(log.DEBUG, "%s request from %s to %s", r.Method, r.RemoteAddr, r.RequestURI)
	if len(r.URL.Path) > 1 {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
	}
	cr.handler.ServeHTTP(w, r)
}

func main() {
	fmt.Println("Start")

	data, _ := ioutil.ReadFile("test.hex")

	ih := intelhex.New()
	err := ih.Load(strings.NewReader(string(data)))
	if err != nil {
		log.Log(log.ERROR, err.Error())
	}

	core := nocan.NewCoreEndpoint()
	core.Topics.Model.Register("foo")
	core.Topics.Model.Register("pizza")

	ports := nocan.NewPortModel()
	ports.Add(&nocan.LogEndpoint{})
	ports.Add(core)

	router := httprouter.New()
	router.GET("/topic", core.Topics.Index)
	router.GET("/topic/*topic", core.Topics.Show)
	router.PUT("/topic/*topic", core.Topics.Update)
	//http.Handle("/api/topic/", &TopicHandler{Topics: tm})
	//http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/Users/apannetrat/go/src/pannetrat.com/nocan/static/"))))
	//http.HandleFunc("/", defaultHandler)
	go ports.ListenAndServe()
	http.ListenAndServe(":8888", &CheckRouter{router})
	fmt.Println("Done")
}

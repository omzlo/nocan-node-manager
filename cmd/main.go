package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"pannetrat.com/nocan"
)

type TopicHandler struct {
	Topics *nocan.TopicManager
}

func (th *TopicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topicName := r.URL.Path[len("/topic/"):]
	topic := th.Topics.FindByName(topicName)

	switch r.Method {
	case "GET":
		if topic < 0 {
			http.Error(w, "Topic does not exist.", http.StatusNotFound)
		} else {
			content, _ := th.Topics.GetContent(topic)
			w.Write(content)
		}
	case "PUT":
		if topic < 0 {
			http.Error(w, "Topic does not exist.", http.StatusNotFound)
		} else {
			body, _ := ioutil.ReadAll(r.Body)
			th.Topics.SetContent(topic, body)
		}
	default:
		http.Error(w, "Only GET and PUT are allowed.", http.StatusMethodNotAllowed)
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I hate %s!", r.URL.Path[1:])
}

func main() {
	fmt.Println("Start")
	tm := nocan.NewTopicManager()
	tm.Register("pizza")
	tm.Register("foo")
	http.Handle("/topic/", &TopicHandler{Topics: tm})
	http.HandleFunc("/", defaultHandler)
	http.ListenAndServe(":8888", nil)
	fmt.Println("Done")
}

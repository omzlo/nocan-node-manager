package nocan

import (
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"strings"
)

type TopicController struct {
	Model    *TopicModel
	Endpoint *CoreEndpoint
}

func RenderJSON(w http.ResponseWriter, v interface{}) bool {
	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if _, err := w.Write(js); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func NewTopicController(endpoint *CoreEndpoint) *TopicController {
	return &TopicController{NewTopicModel(), endpoint}
}

func (tc *TopicController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []string

	tc.Model.Each(func(topic Topic, state *TopicState, _ interface{}) {
		res = append(res, state.Name)
	}, nil)

	RenderJSON(w, res)
}

func TrimLeftSlash(s string) string {
	return strings.TrimPrefix(s, "/")
}

func (tc *TopicController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	topicName := TrimLeftSlash(params.ByName("topic"))

	topic := tc.Model.FindByName(topicName)

	if topic < 0 {
		http.Error(w, "Topic does not exist", http.StatusNotFound)
		return
	}
	content, _ := tc.Model.GetContent(topic)
	w.Write(content)
}

func (tc *TopicController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
    topicName := TrimLeftSlash(params.ByName("topic"))

    topic := tc.Model.FindByName(topicName)
	
    if topic < 0 {
		http.Error(w, "Topic " + topicName + " does not exist", http.StatusNotFound)
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	tc.Model.SetContent(topic, body)
	tc.Endpoint.Publish(0, topic, body)
}

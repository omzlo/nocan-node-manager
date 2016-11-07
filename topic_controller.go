package nocan

import (
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"pannetrat.com/nocan/model"
	"strings"
)

type TopicController struct {
	Model       *model.TopicModel
	TaskManager *model.TaskManager
}

func NewTopicController(manager *model.TaskManager) *TopicController {
	return &TopicController{model.NewTopicModel(), manager}
}

func (tc *TopicController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []string

	tc.Model.Each(func(topic model.Topic, state *model.TopicState, _ interface{}) {
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
		http.Error(w, "Topic "+topicName+" does not exist", http.StatusNotFound)
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	tc.Model.SetContent(topic, body)
	tc.TaskManager.CreateAndLaunchTaskFunction("publish", func(state *model.TaskState) {
		state.Publish(0, topic, body)
	})
}

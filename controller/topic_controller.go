package controller

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"pannetrat.com/nocan/view"
	"strings"
)

type TopicController struct {
	Application *Application
	Port        *model.Port
	Model       *model.TopicModel
}

func NewTopicController(app *Application) *TopicController {
	return &TopicController{Application: app, Port: app.PortManager.CreatePort("topics"), Model: model.NewTopicModel()}
}

func (tc *TopicController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []string

	tc.Model.Each(func(topic model.Topic, state *model.TopicState, _ interface{}) {
		res = append(res, state.Name)
	}, nil)

	context := view.NewContext(r, res)

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "topic_index", context)
	}
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

	context := view.NewContext(r, string(content))

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "topic_show", context)
	}
}

func (tc *TopicController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	topicName := TrimLeftSlash(params.ByName("topic"))

	topic := tc.Model.FindByName(topicName)

	if topic < 0 {
		http.Error(w, "Topic "+topicName+" does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	value := []byte(r.Form.Get("value"))
	tc.Model.SetContent(topic, value)
	tc.Port.SendMessage(model.NewPublishMessage(0, topic, value))

	if !AcceptJSON(r) {
		context := view.NewContext(r, nil)
		context.AddFlashItem("notice", "Successfully updated topic")
		view.RedirectTo(w, r, fmt.Sprintf("/api/topics/%s", topicName), context)
	}
}

func (tc *TopicController) Run() {
	for {
		m := <-tc.Port.Input

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_TOPIC_REGISTER:
				var topic_id model.Topic
				var err error

				topic_expanded, ok := tc.Application.Nodes.Model.ExpandKeywords(m.Id.GetNode(), string(m.Data))
				if ok {
					topic_id, err = tc.Model.Register(topic_expanded)
					if err != nil {
						clog.Warning("NOCAN_SYS_TOPIC_REGISTER: Failed to register topic %s (expanded from %s) for node %d, %s", topic_expanded, string(m.Data), m.Id.GetNode(), err.Error())
					} else {
						clog.Info("NOCAN_SYS_TOPIC_REGISTER: Registered topic %s for node %d as %d", topic_expanded, m.Id.GetNode(), topic_id)
					}
				} else {
					topic_id = -1
					clog.Warning("NOCAN_SYS_TOPIC_REGISTER: Failed to expand topic name '%s' for node %d", string(m.Data), m.Id.GetNode())
				}
				clog.Warning("TOPIC ID IS %d", topic_id)
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_REGISTER_ACK, uint8(topic_id), nil)
				tc.Port.SendMessage(msg)
			case NOCAN_SYS_TOPIC_LOOKUP:
				var bitmap [8]byte
				if tc.Model.Lookup(string(m.Data), bitmap[:]) {
					clog.Info("NOCAN_SYS_TOPIC_LOOKUP: Node %d succesfully found bitmap for topic %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0, bitmap[:])
					tc.Port.SendMessage(msg)
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_LOOKUP: Node %d failed to find bitmap for topic %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0xFF, nil)
					tc.Port.SendMessage(msg)
				}
			case NOCAN_SYS_TOPIC_UNREGISTER:
				var rval uint8
				if tc.Model.Unregister(model.Topic(m.Id.GetSysParam())) {
					clog.Info("NOCAN_SYS_TOPIC_UNREGISTER: Node %d successfully unregistered topic %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_UNREGISTER: Node %d failed to unregister topic %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0xFF
				}
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_UNREGISTER_ACK, rval, nil)
				tc.Port.SendMessage(msg)
			}
		} else if m.Id.IsPublish() {
			tc.Model.SetContent(m.Id.GetTopic(), m.Data)
		}
	}
}

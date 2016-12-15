package nocan

import (
	"github.com/julienschmidt/httprouter"
	//"io/ioutil"
	"net/http"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"strings"
)

type TopicController struct {
	BaseTask
	Model     *model.TopicModel
	NodeModel *model.NodeModel
	//PortManager *model.PortManager
	//Port        *model.Port
}

func NewTopicController(manager *model.PortManager, nodes *model.NodeModel) *TopicController {
	return &TopicController{Model: model.NewTopicModel(), NodeModel: nodes, BaseTask: BaseTask{manager, manager.CreatePort("topics")}}
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

	r.ParseForm()

	value := []byte(r.Form.Get("value"))
	tc.Model.SetContent(topic, value)
	tc.Port.Publish(0, topic, value)
}

func (tc *TopicController) Run() {
	for {
		m := <-tc.Port.Input

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_TOPIC_REGISTER:
				var topic_id model.Topic
				topic_expanded, ok := tc.NodeModel.ExpandKeywords(m.Id.GetNode(), string(m.Data))
				if ok {
					topic_id, err := tc.Model.Register(topic_expanded)
					if err != nil {
						clog.Warning("NOCAN_SYS_TOPIC_REGISTER: Failed to register topic %s (expanded from %s) for node %d, %s", topic_expanded, string(m.Data), m.Id.GetNode(), err.Error())
					} else {
						clog.Info("NOCAN_SYS_TOPIC_REGISTER: Registered topic %s for node %d as %d", topic_expanded, m.Id.GetNode(), topic_id)
					}
				} else {
					topic_id = -1
					clog.Warning("NOCAN_SYS_TOPIC_REGISTER: Failed to expand topic name '%s' for node %d", string(m.Data), m.Id.GetNode())
				}
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
		}
	}
}

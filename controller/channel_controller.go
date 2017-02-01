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

type ChannelController struct {
	Application *Application
	Port        *model.Port
	Model       *model.ChannelModel
}

func NewChannelController(app *Application) *ChannelController {
	return &ChannelController{Application: app, Port: app.PortManager.CreatePort("channels"), Model: model.NewChannelModel()}
}

func (tc *ChannelController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []string

	tc.Model.Each(func(channel model.Channel, state *model.ChannelState, _ interface{}) {
		res = append(res, state.Name)
	}, nil)

	context := view.NewContext(r, res)

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "channel_index", context)
	}
}

func TrimLeftSlash(s string) string {
	return strings.TrimPrefix(s, "/")
}

func (tc *ChannelController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	channelName := TrimLeftSlash(params.ByName("channel"))

	channel := tc.Model.FindByName(channelName)

	if channel < 0 {
		http.Error(w, "Channel does not exist", http.StatusNotFound)
		return
	}
	content, _ := tc.Model.GetContent(channel)

	context := view.NewContext(r, string(content))

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "channel_show", context)
	}
}

func (tc *ChannelController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	channelName := TrimLeftSlash(params.ByName("channel"))

	channel := tc.Model.FindByName(channelName)

	if channel < 0 {
		http.Error(w, "Channel "+channelName+" does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	value := []byte(r.Form.Get("value"))
	tc.Model.SetContent(channel, value)
	tc.Port.SendMessage(model.NewPublishMessage(0, channel, value))

	if !AcceptJSON(r) {
		context := view.NewContext(r, nil)
		context.AddFlashItem("notice", "Successfully updated channel")
		view.RedirectTo(w, r, fmt.Sprintf("/api/channels/%s", channelName), context)
	}
}

func (tc *ChannelController) Run() {
	for {
		m := <-tc.Port.Input

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_CHANNEL_REGISTER:
				var channel_id model.Channel
				var err error

				channel_expanded, ok := tc.Application.Nodes.Model.ExpandKeywords(m.Id.GetNode(), string(m.Data))
				if ok {
					channel_id, err = tc.Model.Register(channel_expanded)
					if err != nil {
						clog.Warning("NOCAN_SYS_CHANNEL_REGISTER: Failed to register channel %s (expanded from %s) for node %d, %s", channel_expanded, string(m.Data), m.Id.GetNode(), err.Error())
					} else {
						clog.Info("NOCAN_SYS_CHANNEL_REGISTER: Registered channel %s for node %d as %d", channel_expanded, m.Id.GetNode(), channel_id)
					}
				} else {
					channel_id = -1
					clog.Warning("NOCAN_SYS_CHANNEL_REGISTER: Failed to expand channel name '%s' for node %d", string(m.Data), m.Id.GetNode())
				}
				clog.Warning("CHANNEL ID IS %d", channel_id)
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_REGISTER_ACK, uint8(channel_id), nil)
				tc.Port.SendMessage(msg)
			case NOCAN_SYS_CHANNEL_LOOKUP:
				var bitmap [8]byte
				if tc.Model.Lookup(string(m.Data), bitmap[:]) {
					clog.Info("NOCAN_SYS_CHANNEL_LOOKUP: Node %d succesfully found bitmap for channel %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_LOOKUP_ACK, 0, bitmap[:])
					tc.Port.SendMessage(msg)
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_LOOKUP: Node %d failed to find bitmap for channel %s", m.Id.GetNode(), string(m.Data))
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_LOOKUP_ACK, 0xFF, nil)
					tc.Port.SendMessage(msg)
				}
			case NOCAN_SYS_CHANNEL_UNREGISTER:
				var rval uint8
				if tc.Model.Unregister(model.Channel(m.Id.GetSysParam())) {
					clog.Info("NOCAN_SYS_CHANNEL_UNREGISTER: Node %d successfully unregistered channel %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_UNREGISTER: Node %d failed to unregister channel %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0xFF
				}
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_UNREGISTER_ACK, rval, nil)
				tc.Port.SendMessage(msg)
			}
		} else if m.Id.IsPublish() {
			tc.Model.SetContent(m.Id.GetChannel(), m.Data)
		}
	}
}

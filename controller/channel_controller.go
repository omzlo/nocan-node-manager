package controller

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	//"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"pannetrat.com/nocan/view"
	"strings"
)

type ChannelController struct {
}

func NewChannelController() *ChannelController {
	return &ChannelController{}
}

func (tc *ChannelController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []string

	model.Channels.Each(func(channel model.Channel, state *model.ChannelState) {
		res = append(res, state.Name)
	})

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

	channel := model.Channels.FindByName(channelName)

	if channel < 0 {
		http.Error(w, "Channel does not exist", http.StatusNotFound)
		return
	}
	content, _ := model.Channels.GetContent(channel)

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

	channel := model.Channels.FindByName(channelName)

	if channel < 0 {
		http.Error(w, "Channel "+channelName+" does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	value := []byte(r.Form.Get("value"))
	model.Channels.Publish(channel, value)

	if !AcceptJSON(r) {
		context := view.NewContext(r, nil)
		context.AddFlashItem("notice", "Successfully updated channel")
		view.RedirectTo(w, r, fmt.Sprintf("/api/channels/%s", channelName), context)
	}
}

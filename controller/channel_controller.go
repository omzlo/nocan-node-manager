package controller

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	//"pannetrat.com/nocan/clog"
	"encoding/hex"
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

	channel, ok := model.Channels.Lookup(channelName)

	if !ok {
		view.LogHttpError(w, "Channel does not exist", http.StatusNotFound)
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

	channel, ok := model.Channels.Lookup(channelName)

	if !ok {
		view.LogHttpError(w, "Channel "+channelName+" does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	var dst []byte
	var err error
	value := r.Form.Get("value")
	if len(value) > 1 && value[0] == '#' {
		if dst, err = hex.DecodeString(value[1:]); err != nil {
			view.LogHttpError(w, "Error decoding hexadecimal string: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		dst = []byte(value)
	}
	model.Channels.Publish(channel, dst)

	if !AcceptJSON(r) {
		context := view.NewContext(r, nil)
		context.AddFlashItem("notice", "Successfully updated channel")
		view.RedirectTo(w, r, fmt.Sprintf("/api/channels/%s", channelName), context)
	}
}

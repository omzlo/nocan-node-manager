package controllers

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/models"
	"pannetrat.com/nocan/view"
	"strconv"
	"time"
)

type InterfaceController struct {
}

func NewInterfaceController() *InterfaceController {
	return &InterfaceController{}
}

func (dc *InterfaceController) GetInterface(interfName string) *models.InterfaceState {
	interf, err := strconv.Atoi(interfName)
	if err != nil {
		return nil
	}
	return models.Interfaces.GetInterface(interf)
}

func (dc *InterfaceController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []int

	models.Interfaces.Each(func(iid int, _ *models.InterfaceState) {
		res = append(res, int(iid))
	})

	context := view.NewContext(r, res)

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "interface_index", context)
	}
}

func (dc *InterfaceController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	interf := dc.GetInterface(params.ByName("interf"))
	if interf == nil {
		view.LogHttpError(w, "Interface does not exist", http.StatusNotFound)
		return
	}

	context := view.NewContext(r, interf)

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "interface_show", context)
	}
}

func (dc *InterfaceController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	interf := dc.GetInterface(params.ByName("interf"))
	if interf == nil {
		view.LogHttpError(w, "Interface does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	var err error

	switch r.Form.Get("c") {
	case "poweron":
		err = interf.DoSetPower(models.INTERFACE_POWER_ON)
	case "poweroff":
		err = interf.DoSetPower(models.INTERFACE_POWER_OFF)
	default:
		view.LogHttpError(w, "missing or incorrect 'c' parameter in request", http.StatusBadRequest)
		return
	}

	if err != nil {
		view.LogHttpError(w, err.Error(), http.StatusServiceUnavailable)
	}

	time.Sleep(1 * time.Second)

	interf.DoRequestPowerStatus()

	/*
		switch {
		case AcceptJSON(r):
			view.RenderJSON(w, view.NewContext(r, "success"))
		default:
			context := view.NewContext(r, nil)
			context.AddFlashItem("notice", "update executed with success")
			view.RedirectTo(w, r, fmt.Sprintf("/api/interfaces/%d", interf.InterfaceId), context)
		}
	*/
	dc.Show(w, r, params)
}

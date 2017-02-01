package controller

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/model"
	"pannetrat.com/nocan/view"
	"strconv"
)

type InterfaceController struct {
	Application *Application
	Port        *model.Port
	Model       *model.InterfaceModel
}

func NewInterfaceController(app *Application) *InterfaceController {
	return &InterfaceController{Application: app, Port: app.PortManager.CreatePort("interf"), Model: model.NewInterfaceModel()}
}

func (dc *InterfaceController) GetInterface(interfName string) *model.Interface {
	interf, err := strconv.Atoi(interfName)
	if err != nil {
		return nil
	}
	return dc.Model.GetInterface(model.InterfaceId(interf))
}

func (dc *InterfaceController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []int

	for i, _ := range dc.Model.Interfaces {
		res = append(res, int(i))
	}

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

	var req *model.SerialCanRequest
	var err error

	switch r.Form.Get("c") {
	case "poweron":
		req, err = interf.SendSetPower(model.INTERFACE_POWER_ON)
	case "poweroff":
		req, err = interf.SendSetPower(model.INTERFACE_POWER_OFF)
	default:
		view.LogHttpError(w, "missing or incorrect 'c' parameter in request", http.StatusBadRequest)
		return
	}

	if err != nil {
		view.LogHttpError(w, err.Error(), http.StatusServiceUnavailable)
	}

	if success := <-req.C; success != model.SERIAL_CAN_REQUEST_STATUS_SUCCESS {
		view.LogHttpError(w, "Command failed", http.StatusInternalServerError)
	}
	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, view.NewContext(r, "success"))
	default:
		context := view.NewContext(r, nil)
		context.AddFlashItem("notice", "update executed with success")
		view.RedirectTo(w, r, fmt.Sprintf("/api/interfaces/%d", interf.InterfaceId), context)
	}
}

func (dc *InterfaceController) Run() {
	dc.Model.Run(dc.Port)
}

/*
type InterfaceControlController struct {
	Interface *InterfaceController
}

func (dcc *InterfaceControlController) Create(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	interf := dcc.GetInterface(params.ByName("interf"))
	if interf==nil {
		view.LogHttpError(w, "Interface does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	var req *model.SerialRequest
	var err error

	select r.PostForm.Get("c") {
	case "poweron":
		req, err = interf.SendSetPower(model.INTERFACE_POWER_ON)
	case "poweroff":
		req, err = interf.SendSetPower(model.INTERFACE_POWER_OFF)
	default:
		view.LogHttpError(w, "missing or incorrect 'c' parameter in form", http.StatusBadRequest)
		return
	}

	if err!=nil {
		view.LogHttpError(w, err.Error(), http.StatusServiceUnavailable)
	}

	if !<-req.C {
		view.LogHttpError(w, "Command failed", http.Status
	}
}
*/

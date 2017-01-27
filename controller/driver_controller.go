package controller

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/model"
	"pannetrat.com/nocan/view"
	"strconv"
)

type DriverController struct {
	Application *Application
	Port        *model.Port
	Model       *model.DriverModel
}

func NewDriverController(app *Application) *DriverController {
	return &DriverController{Application: app, Port: app.PortManager.CreatePort("driver"), Model: model.NewDriverModel()}
}

func (dc *DriverController) GetDriver(driverName string) *model.Driver {
	driver, err := strconv.Atoi(driverName)
	if err != nil {
		return nil
	}
	return dc.Model.GetDriver(model.DriverId(driver))
}

func (dc *DriverController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []int

	for i, _ := range dc.Model.Drivers {
		res = append(res, int(i))
	}

	context := view.NewContext(r, res)

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "driver_index", context)
	}
}

func (dc *DriverController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	driver := dc.GetDriver(params.ByName("driver"))
	if driver == nil {
		view.LogHttpError(w, "Driver does not exist", http.StatusNotFound)
		return
	}

	context := view.NewContext(r, driver)

	switch {
	case AcceptJSON(r):
		view.RenderJSON(w, context)
	default:
		view.RenderAceTemplate(w, "base", "driver_show", context)
	}
}

func (dc *DriverController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	driver := dc.GetDriver(params.ByName("driver"))
	if driver == nil {
		view.LogHttpError(w, "Driver does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	var req *model.SerialCanRequest
	var err error

	switch r.Form.Get("c") {
	case "poweron":
		req, err = driver.SendSetPower(model.DRIVER_POWER_ON)
	case "poweroff":
		req, err = driver.SendSetPower(model.DRIVER_POWER_OFF)
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
		view.RedirectTo(w, r, fmt.Sprintf("/api/drivers/%d", driver.DriverId), context)
	}
}

func (dc *DriverController) Run() {
	dc.Model.Run(dc.Port)
}

/*
type DriverControlController struct {
	Driver *DriverController
}

func (dcc *DriverControlController) Create(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	driver := dcc.GetDriver(params.ByName("driver"))
	if driver==nil {
		view.LogHttpError(w, "Driver does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	var req *model.SerialRequest
	var err error

	select r.PostForm.Get("c") {
	case "poweron":
		req, err = driver.SendSetPower(model.DRIVER_POWER_ON)
	case "poweroff":
		req, err = driver.SendSetPower(model.DRIVER_POWER_OFF)
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

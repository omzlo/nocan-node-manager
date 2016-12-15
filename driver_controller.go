package nocan

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/model"
	"strconv"
)

type DriverController struct {
	BaseTask
	Model *model.DriverModel
}

func NewDriverController(manager *model.PortManager) *DriverController {
	return &DriverController{BaseTask: BaseTask{manager, manager.CreatePort("driver")}, Model: model.NewDriverModel()}
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

	RenderJSON(w, res)
}

func (dc *DriverController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	driver := dc.GetDriver(params.ByName("driver"))
	if driver == nil {
		LogHttpError(w, "Driver does not exist", http.StatusNotFound)
		return
	}
	RenderJSON(w, driver)
}

func (dc *DriverController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	driver := dc.GetDriver(params.ByName("driver"))
	if driver == nil {
		LogHttpError(w, "Driver does not exist", http.StatusNotFound)
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
		LogHttpError(w, "missing or incorrect 'c' parameter in request", http.StatusBadRequest)
		return
	}

	if err != nil {
		LogHttpError(w, err.Error(), http.StatusServiceUnavailable)
	}

	if success := <-req.C; success != model.SERIAL_CAN_REQUEST_STATUS_SUCCESS {
		LogHttpError(w, "Command failed", http.StatusInternalServerError)
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
		LogHttpError(w, "Driver does not exist", http.StatusNotFound)
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
		LogHttpError(w, "missing or incorrect 'c' parameter in form", http.StatusBadRequest)
		return
	}

	if err!=nil {
		LogHttpError(w, err.Error(), http.StatusServiceUnavailable)
	}

	if !<-req.C {
		LogHttpError(w, "Command failed", http.Status
	}
}
*/

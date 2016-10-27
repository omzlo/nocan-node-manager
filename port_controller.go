package nocan

import (
	"github.com/julienschmidt/httprouter"
	//"io/ioutil"
	"net/http"
	//		"strings"
)

type PortController struct {
	Model       *PortModel
	Application *ApplicationController
}

func NewPortController(app *ApplicationController) *PortController {
	return &PortController{NewPortModel(), app}
}

func (pc *PortController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []int

	pc.Model.Each(func(port Port, _ *PortState, _ interface{}) {
		res = append(res, int(port))
	}, nil)

	RenderJSON(w, res)
}

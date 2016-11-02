package nocan

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/model"
)

type PortController struct {
	Model       *model.PortModel
	Application *ApplicationController
}

func NewPortController(app *ApplicationController) *PortController {
	return &PortController{model.NewPortModel(), app}
}

func (pc *PortController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []int

	pc.Model.Each(func(port model.Port, _ *model.PortState, _ interface{}) {
		res = append(res, int(port))
	}, nil)

	RenderJSON(w, res)
}

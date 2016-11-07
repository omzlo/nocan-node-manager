package nocan

/*
import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/model"
)

type SerialController struct {
	Model    *model.SerialModel
	MainTask *model.Task
}

func NewSerialController(app *ApplicationController) *SerialController {
	return &SerialController{model.NewSerialModel(), app}
}

func (pc *SerialController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []int

	pc.Model.Each(func(port model.Serial, _ *model.SerialState, _ interface{}) {
		res = append(res, int(port))
	}, nil)

	RenderJSON(w, res)
}
*/

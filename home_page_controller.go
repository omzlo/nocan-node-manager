package nocan

import (
	"github.com/eknkc/amber"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type HomePageController struct {
}

func NewHomePageController() *HomePageController {
	return &HomePageController{}
}

func (hc *HomePageController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tpl, err := amber.CompileFile("../templates/main.amber", amber.Options{true, false})
	if err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tpl.Execute(w, map[string]string{"Msg": "Hello Ace"}); err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

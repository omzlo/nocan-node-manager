package controller

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/view"
)

type HomePageController struct {
}

func NewHomePageController() *HomePageController {
	return &HomePageController{}
}

func (hc *HomePageController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	view.RenderAceTemplate(w, "base", "main", view.NewContext(r, nil))
	/*
		tpl, err := amber.CompileFile("../templates/main.amber", amber.Options{true, false})
		if err != nil {
			view.LogHttpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tpl.Execute(w, map[string]string{"Msg": "Hello Ace"}); err != nil {
			view.LogHttpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	*/
}

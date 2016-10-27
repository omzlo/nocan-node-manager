package nocan

import (
	"github.com/eknkc/amber"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

type NodePageController struct {
	Model *NodeModel
}

func NewNodePageController() *NodePageController {
	return &NodePageController{}
}

//func (nc *NodePageController) RenderNode()

func (nc *NodePageController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tpl, err := amber.CompileFile("../templates/nodes.amber", amber.Options{true, false})
	if err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tpl.Execute(w, map[string]string{"Msg": "Hello Ace"}); err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

/*
func (nc *NodePageController) Show(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	node := param.ByName("node")
	if !ok {

	}
}
*/

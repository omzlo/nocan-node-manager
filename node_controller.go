package nocan

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/model"
	"strconv"
)

type NodeController struct {
	Model       *model.NodeModel
	Application *ApplicationController
}

func NewNodeController(app *ApplicationController) *NodeController {
	return &NodeController{model.NewNodeModel(), app}
}

func (nc *NodeController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []string

	nc.Model.Each(func(_ model.Node, state *model.NodeState, _ interface{}) {
		res = append(res, model.UidToString(state.Uid[:]))
	}, nil)

	RenderJSON(w, res)
}

func (nc *NodeController) GetNode(nodeName string) (model.Node, bool) {
	if len(nodeName) > 3 {
		var uid [8]byte
		if err := model.StringToUid(nodeName, uid[:]); err != nil {
			return model.Node(-1), false
		}
		return nc.Model.ByUid(uid)
	}

	node, err := strconv.Atoi(nodeName)

	if err != nil {
		return model.Node(-1), false
	}

	return model.Node(node), true
}

func (nc *NodeController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, ok := nc.GetNode(params.ByName("node"))
	if !ok {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}

	props := nc.Model.GetProperties(node)
	if props == nil {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}

	RenderJSON(w, props)
}

/*
func (tc *NodeController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

}
*/

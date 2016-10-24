package nocan

import (
    "strconv"
    "net/http"
    "github.com/julienschmidt/httprouter"
)

type NodeController struct {
    Model *NodeModel
    Endpoint *CoreEndpoint
}

func NewNodeController(endpoint *CoreEndpoint) *NodeController {
    return &NodeController{NewNodeModel(), endpoint}
}

func (nc *NodeController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    var res []int

	nc.Model.Each(func(node Node, _ *NodeState, _ interface{}) {
		res = append(res, int(node))
	}, nil)

	RenderJSON(w, res)
}

func (nc *NodeController) GetNode(nodeName string) (Node, bool) {
    if len(nodeName)>3 {
        var uid [8]byte
        if err := StringToUid(nodeName,uid[:]); err!=nil {
            return Node(-1), false
        }
        return nc.Model.ByUid(uid)
    }

    node, err := strconv.Atoi(nodeName)

    if err!=nil {
        return Node(-1), false
    }

    return Node(node), true
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

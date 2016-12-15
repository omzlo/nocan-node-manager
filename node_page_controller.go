package nocan

import (
	"encoding/json"
	"fmt"
	//"github.com/eknkc/amber"
	"github.com/julienschmidt/httprouter"
	"github.com/yosssi/ace"
	"net/http"
	//"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
)

type NodePageController struct {
	Model *model.NodeModel
}

func NewNodePageController() *NodePageController {
	return &NodePageController{}
}

//func (nc *NodePageController) RenderNode()

func GetJson(r *http.Request, path string, result interface{}) error {
	url := "http://" + r.Host + path

	req, err := http.Get(url)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	return json.NewDecoder(req.Body).Decode(result)
}

func (nc *NodePageController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	tpl, err := ace.Load("base", "nodes", &ace.Options{BaseDir: "../templates", Indent: "  ", DynamicReload: true})
	if err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var nodearray []int
	if err := GetJson(r, "/api/nodes", &nodearray); err != nil {
		LogHttpError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodes := make([]interface{}, len(nodearray)-1)

	for k, v := range nodearray {
		if k > 0 {
			err := GetJson(r, fmt.Sprintf("/api/nodes/%d", v), &nodes[k-1])
			if err != nil {
				LogHttpError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tpl.Execute(w, map[string]interface{}{"Nodes": nodes}); err != nil {
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

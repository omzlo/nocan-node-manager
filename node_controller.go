package nocan

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"strconv"
)

type NodeController struct {
	Model       *model.NodeModel
	TaskManager *model.TaskManager
	Flash       NodeFlashController
}

func NewNodeController(manager *model.TaskManager) *NodeController {
	controller := &NodeController{Model: model.NewNodeModel(), TaskManager: manager}
	controller.Flash.ParentController = controller
	return controller
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
	clog.Debug("NODE=%d %b", node, ok)
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

type NodeFlashController struct {
	// For now, we will try to keep things simple by allowing only one firmware to be accessed simultaneously
	// later, we whould integrate this in the node model.
	ParentController *NodeController
	Inprogress       bool
	Done			 chan bool
	Hex				 *intelhex.IntelHex
}

const (
	SPM_PAGE_SIZE = 128
	READ_SIZE     = 2048
)


func (nfc *NodeFlashController) DownloadFlashTask(state *model.TaskState) {
	var addr uint16
	var i uint16
	var data [8]byte

	for i=0; i<READ_SIZE/SPM_PAGE_SIZE;i++ {
		address = i*SPM_PAGE_SIZE
		data[0]=0
		data[1]=0
		data[2]=byte(address>>8)
		data[3]=byte(address&0xFF)
		m := model.NewSystemMessage(0,NOCAN_SYS_BOOTLOADER_SET_ADDRESS),'F', data[:4])
		state.SendMessage(m)
		
	}

}


func (nfc *NodeFlashController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	_, ok := nfc.ParentController.GetNode(params.ByName("node"))
	if !ok {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}
	if nfc.Inprogress {
		LogHttpError(w, "Flash upload or download already in progress", http.StatusConflict)
		return
	} else {
		// should we mutex this?
		nfc.Inprogress = true
	}
	//LogHttpError(w, "Flash download is not implemeneted yet", http.StatusNotImplemented)
	
	nfc.Hex = intelhex.New()
	nfc.Done = make(chan bool)
	

}

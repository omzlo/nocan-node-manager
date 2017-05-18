package controllers

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/intelhex"
	"pannetrat.com/nocan/models"
	"pannetrat.com/nocan/view"
	"strconv"
	"strings"
)

type NodeController struct {
}

func NewNodeController() *NodeController {
	controller := &NodeController{}
	return controller
}

func (nc *NodeController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []models.Node = make([]models.Node, 0)

	models.Nodes.Each(func(n models.Node, _ *models.NodeState) {
		res = append(res, n)
	})

	context := view.NewContext(r, res)

	if AcceptJSON(r) {
		view.RenderJSON(w, context)
	} else {
		view.RenderAceTemplate(w, "base", "node_index", context)
	}
}

func (nc *NodeController) GetNode(nodeName string) (models.Node, bool) {
	if len(nodeName) > 3 {
		var uid [8]byte
		if err := models.StringToUdid(nodeName, uid[:]); err != nil {
			return models.Node(-1), false
		}
		return models.Nodes.ByUdid(uid)
	}

	node, err := strconv.Atoi(nodeName)

	if err != nil {
		return models.Node(-1), false
	}

	return models.Node(node), true
}

func (nc *NodeController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, ok := nc.GetNode(params.ByName("node"))
	if !ok {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}

	props := models.Nodes.GetProperties(node)
	if props == nil {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}

	context := view.NewContext(r, props)

	if AcceptJSON(r) {
		view.RenderJSON(w, context)
	} else {
		view.RenderAceTemplate(w, "base", "node_show", context)
	}
}

func (nc *NodeController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, ok := nc.GetNode(params.ByName("node"))
	if !ok {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}

	if models.Nodes.GetProperties(node) == nil {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	/* TODO: add JSON/HTTP processing */

	switch r.Form.Get("c") {
	case "reboot":
		if err := models.Nodes.DoReboot(node); err != nil {
			view.LogHttpError(w, err.Error(), http.StatusServiceUnavailable)
		}
	case "ping":
		if err := models.Nodes.DoPing(node); err != nil {
			view.LogHttpError(w, err.Error(), http.StatusServiceUnavailable)
		}
	default:
		view.LogHttpError(w, "Unknown command", http.StatusBadRequest)
	}
	/* success */
}

func (nc *NodeController) GetFirmwareNodeAndType(w http.ResponseWriter, r *http.Request, params httprouter.Params) (models.Node, byte, bool) {
	node, ok := nc.GetNode(params.ByName("node"))
	if !ok {
		view.LogHttpError(w, "Node does not exist", http.StatusNotFound)
		return 0, 0, false
	}
	if node == 0 {
		view.LogHttpError(w, "Node 0 firmware cannot be accessed", http.StatusNotFound)
		return 0, 0, false
	}

	var fwtype byte
	if strings.HasSuffix(r.URL.Path, "/flash") {
		fwtype = 'F'
	} else if strings.HasSuffix(r.URL.Path, "/eeprom") {
		fwtype = 'E'
	} else {
		// should never get here
		view.LogHttpError(w, "Not found", http.StatusNotFound)
		return 0, 0, false
	}
	return node, fwtype, true
}

func (nc *NodeController) ShowFirmware(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, fwtype, ok := nc.GetFirmwareNodeAndType(w, r, params)
	if !ok {
		return
	}

	var fwsize uint32
	fwsize_string := r.URL.Query().Get("size")
	if len(fwsize_string) == 0 {
		if fwtype == 'F' {
			fwsize = 0x7000
		} else {
			fwsize = 0x400
		}
	} else {
		fwsize64, err := strconv.ParseUint(fwsize_string, 10, 32)
		if err != nil {
			view.LogHttpError(w, "Incorrect size parameter", http.StatusBadRequest)
			return
		}
		if fwtype == 'F' && fwsize64 > 0x7000 {
			view.LogHttpError(w, "Flash size cannot exceed 28672 bytes (the following 4K above this limit is used by the bootloader)", http.StatusBadRequest)
		}
		if fwtype == 'E' && fwsize64 > 0x400 {
			view.LogHttpError(w, "Eeprom size cannot exceed 1024 bytes", http.StatusBadRequest)
			return
		}
		fwsize = uint32(fwsize64)
	}

	jobid := models.Jobs.CreateJob(func(state *models.JobState) {
		models.Nodes.DownloadFirmware(state, node, fwtype, fwsize)
	})

	w.Header().Set("Location", fmt.Sprintf("/api/jobs/%d", jobid))
	w.WriteHeader(http.StatusAccepted)
}

func (nc *NodeController) CreateFirmware(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, fwtype, ok := nc.GetFirmwareNodeAndType(w, r, params)
	if !ok {
		return
	}

	r.ParseMultipartForm(1 << 20)
	file, header, err := r.FormFile("firmware")
	if err != nil {
		view.LogHttpError(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	ihex := intelhex.New()
	if err := ihex.Load(file); err != nil {
		view.LogHttpError(w, "Failed to parse firmware: "+err.Error(), http.StatusBadRequest)
		return
	}

	clog.Debug("Uploaded firmware '%s' is %d bytes", header.Filename, ihex.Size)

	jobid := models.Jobs.CreateJob(func(state *models.JobState) {
		models.Nodes.UploadFirmware(state, node, fwtype, ihex)
	})

	w.Header().Set("Location", fmt.Sprintf("/api/jobs/%d", jobid))
	w.WriteHeader(http.StatusAccepted)
}

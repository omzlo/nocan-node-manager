package controller

import (
	"bytes"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/bitmap"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/intelhex"
	"pannetrat.com/nocan/model"
	"pannetrat.com/nocan/view"
	"strconv"
	"strings"
	"sync/atomic"
)

type NodeController struct {
	Application *Application
	Port        *model.Port
	Model       *model.NodeModel
	Firmware    NodeFirmwareController
}

func NewNodeController(app *Application, nodeinfo string) *NodeController {
	controller := &NodeController{Port: app.PortManager.CreatePort("nodes"), Application: app, Model: model.NewNodeModel()}
	if err := controller.Model.LoadFromFile(nodeinfo); err != nil {
		clog.Warning("Could not load node information form %s: %s", nodeinfo, err.Error())
	}
	controller.Firmware.Application = app
	return controller
}

func (nc *NodeController) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var res []model.Node = make([]model.Node, 0)

	nc.Model.Each(func(n model.Node, _ *model.NodeState, _ interface{}) {
		res = append(res, n)
	}, nil)

	context := view.NewContext(r, res)

	if AcceptJSON(r) {
		view.RenderJSON(w, context)
	} else {
		view.RenderAceTemplate(w, "base", "node_index", context)
	}
}

func (nc *NodeController) GetNode(nodeName string) (model.Node, bool) {
	if len(nodeName) > 3 {
		var uid [8]byte
		if err := model.StringToUdid(nodeName, uid[:]); err != nil {
			return model.Node(-1), false
		}
		return nc.Model.ByUdid(uid)
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

	if nc.Model.GetProperties(node) == nil {
		http.Error(w, "Node does not exist", http.StatusNotFound)
		return
	}

	r.ParseForm()

	port := nc.Application.PortManager.CreatePort("update-node")
	defer nc.Application.PortManager.DestroyPort(port)

	/* TODO: add JSON/HTTP processing */

	switch r.Form.Get("c") {
	case "reboot":
		port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_NODE_BOOT_REQUEST, 0, nil))
		if port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_NODE_BOOT_ACK), model.DEFAULT_TIMEOUT) == nil {
			view.LogHttpError(w, "Node could not be rebooted", http.StatusServiceUnavailable)
			return
		}
	case "ping":
		port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_NODE_PING, 0, nil))
		if port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_NODE_PING_ACK), model.DEFAULT_TIMEOUT) == nil {
			view.LogHttpError(w, "Node could not be pinged", http.StatusServiceUnavailable)
			return
		}
	default:
		view.LogHttpError(w, "Unknown command", http.StatusBadRequest)
		return
	}
}

func (nc *NodeController) Run() {
	for {
		m := <-nc.Port.Input

		nc.Model.Touch(m.Id.GetNode())

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_ADDRESS_REQUEST:
				node_id, err := nc.Model.Register(m.Data)
				if err != nil {
					clog.Warning("NOCAN_SYS_ADDRESS_REQUEST: Failed to register %s, %s", model.UdidToString(m.Data), err.Error())
				} else {
					clog.Info("NOCAN_SYS_ADDRESS_REQUEST: Registered %s as node %d", model.UdidToString(m.Data), node_id)
				}
				msg := model.NewSystemMessage(0, NOCAN_SYS_ADDRESS_CONFIGURE, uint8(node_id), m.Data)
				nc.Port.SendMessage(msg)
			case NOCAN_SYS_ADDRESS_CONFIGURE_ACK:
				// TODO
			case NOCAN_SYS_ADDRESS_LOOKUP:
				node_id, _ := nc.Model.Lookup(m.Data)
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_ADDRESS_LOOKUP_ACK, uint8(node_id), m.Data)
				nc.Port.SendMessage(msg)
			case NOCAN_SYS_CHANNEL_SUBSCRIBE:
				if nc.Model.Subscribe(m.Id.GetNode(), m.Data) {
					clog.Info("NOCAN_SYS_CHANNEL_SUBSCRIBE: Node %d successfully subscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_SUBSCRIBE: Node %d failed to subscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				}
			case NOCAN_SYS_CHANNEL_UNSUBSCRIBE:
				if nc.Model.Unsubscribe(m.Id.GetNode(), m.Data) {
					clog.Info("NOCAN_SYS_CHANNEL_UNSUBSCRIBE: Node %d successfully unsubscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_UNSUBSCRIBE: Node %d failed to unsubscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				}
			}
		}
	}
}

type NodeFirmwareController struct {
	// For now, we will try to keep things simple by allowing only one firmware to be accessed simultaneously
	// later, we whould integrate this in the node model.
	Application *Application
	Inprogress  int32
}

const (
	SPM_PAGE_SIZE = 128
	READ_SIZE     = 2048
)

func (nfc *NodeFirmwareController) DownloadFirmware(state *model.JobState, node model.Node, memtype byte, memlength uint32) bool {
	var address uint32
	var i uint32
	var data [8]byte

	port := nfc.Application.PortManager.CreatePort("firmware-download")
	defer nfc.Application.PortManager.DestroyPort(port)

	port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_NODE_BOOT_REQUEST, 0, nil))

	if port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_NODE_BOOT_ACK), model.EXTENDED_TIMEOUT) == nil {
		err := fmt.Errorf("NOCAN_SYS_NODE_BOOT_ACK failed for node %d", node)
		state.UpdateStatus(model.JobFailed, err)
		clog.Error(err.Error())
		return false
	}

	ihex := intelhex.New()

	for i = 0; i < memlength/SPM_PAGE_SIZE; i++ {
		address = i * SPM_PAGE_SIZE
		data[0] = 0
		data[1] = 0
		data[2] = byte(address >> 8)
		data[3] = byte(address & 0xFF)
		port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS, memtype, data[:4]))
		if port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS_ACK), model.DEFAULT_TIMEOUT) == nil {
			err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_SET_ADDRESS failed for node %d at address=0x%x", node, address)
			state.UpdateStatus(model.JobFailed, err)
			clog.Error(err.Error())
			return false
		}
		for pos := 0; pos < SPM_PAGE_SIZE; pos += 8 {
			port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_READ, 8, nil))
			response := port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_READ_ACK), model.DEFAULT_TIMEOUT)
			if response == nil {
				err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_READ failed for node %d at address=0x%x", node, address)
				state.UpdateStatus(model.JobFailed, err)
				clog.Error(err.Error())
				return false
			}
			ihex.Add(0, address, response.Data)
			address += 8
		}
		state.UpdateProgress(uint(address * 100 / memlength))
	}
	var buf bytes.Buffer
	ihex.Save(&buf)
	state.Result = buf.Bytes()
	state.UpdateProgress(100)
	state.UpdateStatus(model.JobCompleted, nil)
	return true
}

func (nfc *NodeFirmwareController) UploadFirmware(state *model.JobState, node model.Node, memtype byte, ihex *intelhex.IntelHex) bool {
	var address uint32
	var data [8]byte

	port := nfc.Application.PortManager.CreatePort("firmware-upload")
	defer nfc.Application.PortManager.DestroyPort(port)

	port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_NODE_BOOT_REQUEST, 0, nil))

	if port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_NODE_BOOT_ACK), model.EXTENDED_TIMEOUT) == nil {
		err := fmt.Errorf("NOCAN_SYS_NODE_BOOT_ACK failed for node %d", node)
		state.UpdateStatus(model.JobFailed, err)
		clog.Error(err.Error())
		return false
	}

	for _, block := range ihex.Blocks {
		blocksize := uint32(len(block.Data))

		for page_offset := uint32(0); page_offset < blocksize; page_offset += SPM_PAGE_SIZE {
			base_address := block.Address + page_offset
			data[0] = 0
			data[1] = 0
			data[2] = byte(base_address >> 8)
			data[3] = byte(base_address & 0xFF)
			port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS, memtype, data[:4]))
			if port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS_ACK), model.DEFAULT_TIMEOUT) == nil {
				err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_SET_ADDRESS failed for node %d at address=0x%x", node, address)
				state.UpdateStatus(model.JobFailed, err)
				clog.Error(err.Error())
				return false
			}

			for page_pos := uint32(0); page_pos < SPM_PAGE_SIZE && page_offset+page_pos < blocksize; page_pos += 8 {
				rlen := block.Copy(data[:], page_offset+page_pos, 8)
				port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE, 0, data[:rlen]))
				response := port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_WRITE_ACK), model.DEFAULT_TIMEOUT)
				if response == nil {
					err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_WRITE failed for node %d at address=0x%x", node, address)
					state.UpdateStatus(model.JobFailed, err)
					clog.Error(err.Error())
					return false
				}
			}
			port.SendMessage(model.NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE, 1, nil))
			response := port.WaitForMessage(model.NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_WRITE_ACK), model.DEFAULT_TIMEOUT)
			if response == nil {
				err := fmt.Errorf("Final NOCAN_SYS_BOOTLOADER_WRITE failed for node %d at address=0x%x", node, address)
				state.UpdateStatus(model.JobFailed, err)
				clog.Error(err.Error())
				return false
			}

			state.UpdateProgress(uint((page_offset * 100) / blocksize))
		}
	}
	state.Result = []byte("Uploaded")
	state.UpdateProgress(100)
	state.UpdateStatus(model.JobCompleted, nil)
	return true
}

func (nfc *NodeFirmwareController) GetFirmwareNodeAndType(w http.ResponseWriter, r *http.Request, params httprouter.Params) (model.Node, byte, bool) {
	node, ok := nfc.Application.Nodes.GetNode(params.ByName("node"))
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

func (nfc *NodeFirmwareController) Show(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, fwtype, ok := nfc.GetFirmwareNodeAndType(w, r, params)
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

	if !atomic.CompareAndSwapInt32(&nfc.Inprogress, 0, 1) {
		view.LogHttpError(w, "Firmware upload or download already in progress", http.StatusConflict)
		return
	}
	//view.LogHttpError(w, "Flash download is not implemeneted yet", http.StatusNotImplemented)

	jobid := nfc.Application.Jobs.Model.CreateJob(func(state *model.JobState) {
		nfc.DownloadFirmware(state, node, fwtype, fwsize)
		atomic.StoreInt32(&nfc.Inprogress, 0)
	})

	// http.Redirect(w, r, fmt.Sprintf("/api/jobs/%d", jobid), http.StatusSeeOther)
	w.Header().Set("Location", fmt.Sprintf("/api/jobs/%d", jobid))
	w.WriteHeader(http.StatusAccepted)
}

func (nfc *NodeFirmwareController) Create(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, fwtype, ok := nfc.GetFirmwareNodeAndType(w, r, params)
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

	if !atomic.CompareAndSwapInt32(&nfc.Inprogress, 0, 1) {
		view.LogHttpError(w, "Firmware upload or download already in progress\n", http.StatusConflict)
		return
	}

	jobid := nfc.Application.Jobs.Model.CreateJob(func(state *model.JobState) {
		nfc.UploadFirmware(state, node, fwtype, ihex)
		atomic.StoreInt32(&nfc.Inprogress, 0)
	})

	// http.Redirect(w, r, fmt.Sprintf("/api/jobs/%d", jobid), http.StatusSeeOther)
	w.Header().Set("Location", fmt.Sprintf("/api/jobs/%d", jobid))
	w.WriteHeader(http.StatusAccepted)
}

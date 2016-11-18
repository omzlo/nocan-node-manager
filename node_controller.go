package nocan

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"pannetrat.com/nocan/bitmap"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/intelhex"
	"pannetrat.com/nocan/model"
	"strconv"
	"strings"
	"sync/atomic"
)

type NodeController struct {
	BaseTask
	//PortManager *model.PortManager
	//Port        *model.Port
	Model    *model.NodeModel
	Firmware NodeFirmwareController
}

func NewNodeController(manager *model.PortManager) *NodeController {
	controller := &NodeController{Model: model.NewNodeModel(), BaseTask: BaseTask{manager, manager.CreatePort("nodes")}}
	controller.Firmware.ParentController = controller
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
	clog.Debug("NODE=%d %t", node, ok)
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

func (nc *NodeController) Run() {
	for {
		m := <-nc.Port.Input

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_ADDRESS_REQUEST:
				node_id, err := nc.Model.Register(m.Data)
				if err != nil {
					clog.Warning("NOCAN_SYS_ADDRESS_REQUEST: Failed to register %s, %s", model.UidToString(m.Data), err.Error())
				} else {
					clog.Info("NOCAN_SYS_ADDRESS_REQUEST: Registered %s as node %d", model.UidToString(m.Data), node_id)
				}
				msg := model.NewSystemMessage(0, NOCAN_SYS_ADDRESS_CONFIGURE, uint8(node_id), m.Data)
				nc.Port.SendMessage(msg)
			case NOCAN_SYS_ADDRESS_CONFIGURE_ACK:
				// TODO
			case NOCAN_SYS_ADDRESS_LOOKUP:
				node_id, _ := nc.Model.Lookup(m.Data)
				msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_ADDRESS_LOOKUP_ACK, uint8(node_id), m.Data)
				nc.Port.SendMessage(msg)
			case NOCAN_SYS_TOPIC_SUBSCRIBE:
				if nc.Model.Subscribe(m.Id.GetNode(), m.Data) {
					clog.Info("NOCAN_SYS_TOPIC_SUBSCRIBE: Node %d successfully subscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_SUBSCRIBE: Node %d failed to subscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				}
			case NOCAN_SYS_TOPIC_UNSUBSCRIBE:
				if nc.Model.Unsubscribe(m.Id.GetNode(), m.Data) {
					clog.Info("NOCAN_SYS_TOPIC_UNSUBSCRIBE: Node %d successfully unsubscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				} else {
					clog.Warning("NOCAN_SYS_TOPIC_UNSUBSCRIBE: Node %d failed to unsubscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
				}
			}
		}
	}
}

type NodeFirmwareController struct {
	// For now, we will try to keep things simple by allowing only one firmware to be accessed simultaneously
	// later, we whould integrate this in the node model.
	ParentController *NodeController
	Inprogress       int32
}

const (
	SPM_PAGE_SIZE = 128
	READ_SIZE     = 2048
)

func (nfc *NodeFirmwareController) DownloadFirmware(node model.Node, memtype byte, memlength uint32) *intelhex.IntelHex {
	var address uint32
	var i uint32
	var data [8]byte

	port := nfc.ParentController.PortManager.CreatePort("firmware-download")
	defer nfc.ParentController.PortManager.DestroyPort(port)

	if port.WaitForSystemMessage(node, NOCAN_SYS_NODE_BOOT_ACK, model.EXTENDED_TIMEOUT) == nil {
		clog.Error("NOCAN_SYS_NODE_BOOT_ACK failed for node %d", node)
		return nil
	}

	ihex := intelhex.New()

	for i = 0; i < memlength/SPM_PAGE_SIZE; i++ {
		address = i * SPM_PAGE_SIZE
		data[0] = 0
		data[1] = 0
		data[2] = byte(address >> 8)
		data[3] = byte(address & 0xFF)
		port.SendSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS, memtype, data[:4])
		if port.WaitForSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS_ACK, model.DEFAULT_TIMEOUT) == nil {
			clog.Error("NOCAN_SYS_BOOTLOADER_SET_ADDRESS failed for node %d at address=0x%x", node, address)
			return nil
		}
		for pos := 0; pos < SPM_PAGE_SIZE; pos += 8 {
			port.SendSystemMessage(node, NOCAN_SYS_BOOTLOADER_READ, 8, nil)
			response := port.WaitForSystemMessage(node, NOCAN_SYS_BOOTLOADER_READ_ACK, model.DEFAULT_TIMEOUT)
			if response == nil {
				clog.Error("NOCAN_SYS_BOOTLOADER_READ failed for node %d at address=0x%x", node, address)
				return nil
			}
			ihex.Add(0, address, response.Data)
			address += 8
		}
	}
	return ihex
}

func (nfc *NodeFirmwareController) UploadFirmware(node model.Node, memtype byte, ihex *intelhex.IntelHex) bool {
	var address uint32
	var data [8]byte

	port := nfc.ParentController.PortManager.CreatePort("firmware-upload")
	defer nfc.ParentController.PortManager.DestroyPort(port)

	port.SendSystemMessage(node, NOCAN_SYS_NODE_BOOT_REQUEST, 0, nil)

	if port.WaitForSystemMessage(node, NOCAN_SYS_NODE_BOOT_ACK, model.EXTENDED_TIMEOUT) == nil {
		clog.Error("NOCAN_SYS_NODE_BOOT_ACK failed for node %d", node)
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
			port.SendSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS, memtype, data[:4])
			if port.WaitForSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS_ACK, model.DEFAULT_TIMEOUT) == nil {
				clog.Error("NOCAN_SYS_BOOTLOADER_SET_ADDRESS failed for node %d at address=0x%x", node, address)
				return false
			}

			for page_pos := uint32(0); page_pos < SPM_PAGE_SIZE && page_offset+page_pos < blocksize; page_pos += 8 {
				rlen := block.Copy(data[:], page_offset+page_pos, 8)
				port.SendSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE, 0, data[:rlen])
				response := port.WaitForSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE_ACK, model.DEFAULT_TIMEOUT)
				if response == nil {
					clog.Error("NOCAN_SYS_BOOTLOADER_WRITE failed for node %d at address=0x%x", node, address)
					return false
				}
			}
			port.SendSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE, 1, nil)
			response := port.WaitForSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE_ACK, model.DEFAULT_TIMEOUT)
			if response == nil {
				clog.Error("Final NOCAN_SYS_BOOTLOADER_WRITE failed for node %d at address=0x%x", node, address)
				return false
			}
		}
	}
	return true
}

func (nfc *NodeFirmwareController) GetFirmwareNodeAndType(w http.ResponseWriter, r *http.Request, params httprouter.Params) (model.Node, byte, bool) {
	node, ok := nfc.ParentController.GetNode(params.ByName("node"))
	if !ok {
		LogHttpError(w, "Node does not exist", http.StatusNotFound)
		return 0, 0, false
	}
	if node == 0 {
		LogHttpError(w, "Node 0 firmware cannot be accessed", http.StatusNotFound)
		return 0, 0, false
	}

	var fwtype byte
	if strings.HasSuffix(r.URL.Path, "/flash") {
		fwtype = 'F'
	} else if strings.HasSuffix(r.URL.Path, "/eeprom") {
		fwtype = 'E'
	} else {
		// should never get here
		LogHttpError(w, "Not found", http.StatusNotFound)
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
			LogHttpError(w, "Incorrect size parameter", http.StatusBadRequest)
			return
		}
		if fwtype == 'F' && fwsize64 > 0x7000 {
			LogHttpError(w, "Flash size cannot exceed 28672 bytes (the following 4K above this limit is used by the bootloader)", http.StatusBadRequest)
		}
		if fwtype == 'E' && fwsize64 > 0x400 {
			LogHttpError(w, "Eeprom size cannot exceed 1024 bytes", http.StatusBadRequest)
			return
		}
		fwsize = uint32(fwsize64)
	}

	if !atomic.CompareAndSwapInt32(&nfc.Inprogress, 0, 1) {
		LogHttpError(w, "Firmware upload or download already in progress", http.StatusConflict)
		return
	}
	defer atomic.StoreInt32(&nfc.Inprogress, 0)
	//LogHttpError(w, "Flash download is not implemeneted yet", http.StatusNotImplemented)

	if flash := nfc.DownloadFirmware(node, fwtype, fwsize); flash != nil {
		clog.Info("Successfully downloaded firmware")
		flash.Save(w)
	} else {
		LogHttpError(w, "Failed to download firmware", http.StatusServiceUnavailable)
	}
}

func (nfc *NodeFirmwareController) Update(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	node, fwtype, ok := nfc.GetFirmwareNodeAndType(w, r, params)
	if !ok {
		return
	}
	ihex := intelhex.New()
	if err := ihex.Load(r.Body); err != nil {
		LogHttpError(w, "Failed to upload firmware: "+err.Error(), http.StatusBadRequest)
		return
	}

	clog.Debug("Uploaded firmware is %d bytes", ihex.Size)

	if !atomic.CompareAndSwapInt32(&nfc.Inprogress, 0, 1) {
		LogHttpError(w, "Firmware upload or download already in progress\n", http.StatusConflict)
		return
	}
	defer atomic.StoreInt32(&nfc.Inprogress, 0)

	if nfc.UploadFirmware(node, fwtype, ihex) {
		fmt.Fprintf(w, "Successfully uploaded firmware of type '%c' to node %d\n", fwtype, node)
	}
}

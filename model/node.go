package model

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/intelhex"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

type Node int8

type NodeAttributes map[string]interface{}

type NodeState struct {
	Active        bool             `json:"-"`
	Id            Node             `json:"id"`
	Udid          string           `json:"udid"`
	LastSeen      time.Time        `json:"last_seen"`
	Subscriptions map[Channel]bool `json:"-"`
	Attributes    NodeAttributes   `json:"attributes"`
}

func (ns *NodeState) getStringAttribute(key string) (string, bool) {
	if val, ok := ns.Attributes[key]; ok {
		switch v := val.(type) {
		case string:
			return v, true
		case fmt.Stringer:
			return v.String(), true
		case float64:
			return strconv.FormatFloat(v, 'g', -1, 64), true
		case bool:
			return strconv.FormatBool(v), true
		}
	}
	return "", false
}
func UdidToString(id []byte) string {
	retval := ""

	for i := 0; i < len(id); i++ {
		if i > 0 {
			retval += ":"
		}
		retval += hex.EncodeToString(id[i : i+1])
	}
	return retval
}

func StringToUdid(s string, id []byte) error {
	src := []byte(s)

	if len(id) < 8 {
		return errors.New("Insufficient space to store node uidAttr")
	}

	for i := 0; i < len(s); i += 3 {
		if _, err := hex.Decode(id[i/3:i/3+1], src[i:i+2]); err != nil {
			return err
		}
		if i > 0 && src[i-1] != ':' {
			return errors.New("expected ':' in hex identifier")
		}
	}
	return nil
}

type NodeModel struct {
	Mutex      sync.RWMutex
	States     [128]*NodeState
	Udids      map[string]Node
	NodeFile   string
	Inprogress int32
	Port       *Port
}

func NewNodeModel() *NodeModel {
	return &NodeModel{Udids: make(map[string]Node), Port: PortManager.CreatePort("nodes")}
}

type NodeInfo struct {
	Node       Node           `json:"node"`
	Attributes NodeAttributes `json:"attributes"`
}

func (nm *NodeModel) LoadFromFile(nodefile string) error {
	info := make(map[string]NodeInfo)

	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	nm.NodeFile = nodefile
	data, err := ioutil.ReadFile(nodefile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &info)
	if err != nil {
		clog.Fatal("JSON parsing error in %s: %s", nodefile, err.Error())
	}

	for k, v := range info {
		if nm.States[v.Node] != nil {
			clog.Warning("Node %d appears twice in %s, second instance will be ignored", v.Node, nodefile)
		} else {
			clog.Debug("Pre-registering %s as node %d", k, v.Node)
			nm.States[v.Node] = &NodeState{Active: false, Id: v.Node, Udid: k, Attributes: v.Attributes, Subscriptions: make(map[Channel]bool)}
			nm.Udids[k] = v.Node
		}
	}
	return nil
}

func (nm *NodeModel) SaveToFile() error {
	info := make(map[string]NodeInfo)

	nm.Mutex.RLock()
	defer nm.Mutex.RUnlock()

	for k, v := range nm.Udids {
		info[k] = NodeInfo{Node: v, Attributes: nm.States[v].Attributes}
	}

	js, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(nm.NodeFile, js, 0644)
}

func (nm *NodeModel) ExpandKeywords(node Node, str string) (string, bool) {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	ns := nm.getState(node)
	if ns == nil {
		return "", false
	}

	var rval string
	pos := 0
	for pos < len(str) {
		cval, size := utf8.DecodeRuneInString(str[pos:])
		pos += size
		switch cval {
		case utf8.RuneError:
			return "", false
		case '$':
			cval, size := utf8.DecodeRuneInString(str[pos:])
			pos += size
			switch cval {
			case utf8.RuneError:
				return "", false
			case '$':
				rval += "$"
			case '{':
				keyword := ""
				for {
					cval, size := utf8.DecodeRuneInString(str[pos:])
					pos += size
					if cval == utf8.RuneError {
						return "", false
					}
					if cval == '}' {
						break
					}
					keyword += string(cval)
				}
				if subs, ok := ns.getStringAttribute(keyword); ok {
					rval += subs
				}
			default:
				rval += "$" + string(cval)
			}
		default:
			rval += string(cval)
		}
	}
	return rval, true
}

func (nm *NodeModel) Lookup(node []byte) (Node, bool) {
	if len(node) != 8 {
		return Node(-1), false
	}

	udid := UdidToString(node)

	nm.Mutex.RLock()
	defer nm.Mutex.RUnlock()

	if node, ok := nm.Udids[udid]; ok {
		return node, true
	}
	return Node(-1), false
}

func (nm *NodeModel) Register(node []byte) (Node, error) {
	if len(node) != 8 {
		return Node(-1), errors.New("Node identifier must be 8 bytes long")
	}

	udid := UdidToString(node)

	nm.Mutex.Lock()

	if n, ok := nm.Udids[udid]; ok {
		nm.States[n].Active = true
		nm.Mutex.Unlock()
		return n, nil
	}

	for i := 1; i < 128; i++ {
		if nm.States[i] == nil {
			nm.States[i] = &NodeState{Active: true, Udid: udid, Id: Node(i), Subscriptions: make(map[Channel]bool)}
			nm.Udids[udid] = Node(i)
			nm.Mutex.Unlock()
			if err := nm.SaveToFile(); err != nil {
				clog.Warning("Failed to save node info: %s", err.Error())
			}
			return Node(i), nil
		}
	}

	nm.Mutex.Unlock()
	return Node(-1), errors.New("Maximum number of nodes has been reached.")
}

func (nm *NodeModel) Unregister(node Node) bool {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	ns := nm.getState(node)
	if ns == nil {
		return false
	}
	delete(nm.Udids, ns.Udid)
	nm.States[node] = nil
	return true
}

func (nm *NodeModel) Subscribe(node Node, channel_id Channel) bool {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if ns := nm.getState(node); ns != nil {
		ns.Subscriptions[channel_id] = true
		return true
	}
	return false
}

func (nm *NodeModel) Unsubscribe(node Node, channel_id Channel) bool {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if ns := nm.getState(node); ns != nil {
		delete(ns.Subscriptions, channel_id)
		return true
	}
	return false
}

func (nm *NodeModel) GetProperties(node Node) *NodeState {
	nm.Mutex.RLock()
	defer nm.Mutex.RUnlock()

	if ns := nm.getState(node); ns != nil {
		/*
			props := make(map[string]interface{})

			props["id"] = UdidToString(ns.Udid[:])
			props["last_seen"] = ns.LastSeen.UTC().String()
			props["subscriptions"] = bitmap.Bitmap64ToSlice(ns.Subscriptions[:])
			props["attributes"] = make([]string, 0)
			return props
		*/
		return ns
	}
	return nil
}

func (nm *NodeModel) DoReboot(node Node) error {
	nm.Port.SendMessage(NewSystemMessage(node, NOCAN_SYS_NODE_BOOT_REQUEST, 0x01, nil))
	if nm.Port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_NODE_BOOT_ACK), DEFAULT_TIMEOUT) == nil {
		return fmt.Errorf("Node %d could not be rebooted", node)
	}
	return nil
}

func (nm *NodeModel) DoPing(node Node) error {
	nm.Port.SendMessage(NewSystemMessage(node, NOCAN_SYS_NODE_PING, 0, nil))
	if nm.Port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_NODE_PING_ACK), DEFAULT_TIMEOUT) == nil {
		return fmt.Errorf("Node %d could not be pinged", node)
	}
	return nil
}

func (nm *NodeModel) ByUdid(uid [8]byte) (Node, bool) {
	nm.Mutex.RLock()
	defer nm.Mutex.RUnlock()

	udid := UdidToString(uid[:])
	if n, ok := nm.Udids[udid]; ok {
		return n, true
	}
	return Node(-1), false
}

func (nm *NodeModel) Touch(node Node) {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if ns := nm.getState(node); ns != nil {
		ns.LastSeen = time.Now()
	}
}

func (nm *NodeModel) Each(fn func(Node, *NodeState)) {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	for i := 0; i < 128; i++ {
		if ns := nm.getState(Node(i)); ns != nil {
			fn(Node(i), ns)
		}
	}
}

func (nm *NodeModel) getState(node Node) *NodeState {
	if node < 0 {
		return nil
	}
	if nm.States[node] != nil && nm.States[node].Active {
		return nm.States[node]
	}
	return nil
}

const (
	SPM_PAGE_SIZE = 128
	READ_SIZE     = 2048
)

func (nm *NodeModel) DownloadFirmware(state *JobState, node Node, memtype byte, memlength uint32) error {
	var address uint32
	var i uint32
	var data [8]byte

	clog.Debug("Initiate down")
	if !atomic.CompareAndSwapInt32(&nm.Inprogress, 0, 1) {
		err := fmt.Errorf("Firmware upload or download already in progress, ignoring new request")
		state.UpdateStatus(JobFailed, err)
		return err
	}
	defer atomic.StoreInt32(&nm.Inprogress, 0)

	// If we don't do this and use nm.Port instead, we will conflict with Run()
	port := PortManager.CreatePort("firmware-download")
	defer PortManager.DestroyPort(port)

	port.SendMessage(NewSystemMessage(node, NOCAN_SYS_NODE_BOOT_REQUEST, 0x01, nil))

	if port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_NODE_BOOT_ACK), EXTENDED_TIMEOUT) == nil {
		err := fmt.Errorf("NOCAN_SYS_NODE_BOOT_ACK failed for node %d", node)
		state.UpdateStatus(JobFailed, err)
		return err
	}

	ihex := intelhex.New()

	for i = 0; i < memlength/SPM_PAGE_SIZE; i++ {
		address = i * SPM_PAGE_SIZE
		data[0] = 0
		data[1] = 0
		data[2] = byte(address >> 8)
		data[3] = byte(address & 0xFF)
		port.SendMessage(NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS, memtype, data[:4]))
		if port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS_ACK), DEFAULT_TIMEOUT) == nil {
			err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_SET_ADDRESS failed for node %d at address=0x%x", node, address)
			state.UpdateStatus(JobFailed, err)
			return err
		}
		for pos := 0; pos < SPM_PAGE_SIZE; pos += 8 {
			port.SendMessage(NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_READ, 8, nil))
			response := port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_READ_ACK), DEFAULT_TIMEOUT)
			if response == nil {
				err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_READ failed for node %d at address=0x%x", node, address)
				state.UpdateStatus(JobFailed, err)
				return err
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
	state.UpdateStatus(JobCompleted, nil)
	return nil
}

func (nm *NodeModel) UploadFirmware(state *JobState, node Node, memtype byte, ihex *intelhex.IntelHex) error {
	var address uint32
	var data [8]byte

	if !atomic.CompareAndSwapInt32(&nm.Inprogress, 0, 1) {
		return fmt.Errorf("Firmware upload or download already in progress, ignoring new request")
	}
	defer atomic.StoreInt32(&nm.Inprogress, 0)

	// If we don't do this and use nm.Port instead, we will conflict with Run()
	port := PortManager.CreatePort("firmware-upload")
	defer PortManager.DestroyPort(port)

	port.SendMessage(NewSystemMessage(node, NOCAN_SYS_NODE_BOOT_REQUEST, 0x01, nil))

	if port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_NODE_BOOT_ACK), EXTENDED_TIMEOUT) == nil {
		err := fmt.Errorf("NOCAN_SYS_NODE_BOOT_ACK failed for node %d", node)
		state.UpdateStatus(JobFailed, err)
		return err
	}

	for _, block := range ihex.Blocks {
		blocksize := uint32(len(block.Data))

		for page_offset := uint32(0); page_offset < blocksize; page_offset += SPM_PAGE_SIZE {
			base_address := block.Address + page_offset
			data[0] = 0
			data[1] = 0
			data[2] = byte(base_address >> 8)
			data[3] = byte(base_address & 0xFF)
			port.SendMessage(NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS, memtype, data[:4]))
			if port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_SET_ADDRESS_ACK), DEFAULT_TIMEOUT) == nil {
				err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_SET_ADDRESS failed for node %d at address=0x%x", node, address)
				state.UpdateStatus(JobFailed, err)
				return err
			}

			for page_pos := uint32(0); page_pos < SPM_PAGE_SIZE && page_offset+page_pos < blocksize; page_pos += 8 {
				rlen := block.Copy(data[:], page_offset+page_pos, 8)
				port.SendMessage(NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE, 0, data[:rlen]))
				response := port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_WRITE_ACK), DEFAULT_TIMEOUT)
				if response == nil {
					err := fmt.Errorf("NOCAN_SYS_BOOTLOADER_WRITE failed for node %d at address=0x%x", node, address)
					state.UpdateStatus(JobFailed, err)
					return err
				}
			}
			port.SendMessage(NewSystemMessage(node, NOCAN_SYS_BOOTLOADER_WRITE, 1, nil))
			response := port.WaitForMessage(NewSystemMessageFilter(node, NOCAN_SYS_BOOTLOADER_WRITE_ACK), DEFAULT_TIMEOUT)
			if response == nil {
				err := fmt.Errorf("Final NOCAN_SYS_BOOTLOADER_WRITE failed for node %d at address=0x%x", node, address)
				state.UpdateStatus(JobFailed, err)
				return err
			}
			state.UpdateProgress(uint((page_offset * 100) / blocksize))
		}
	}
	state.Result = nil
	state.UpdateProgress(100)
	state.UpdateStatus(JobCompleted, nil)
	return nil
}

func (nm *NodeModel) Run() {
	for {
		m := <-nm.Port.Input

		nm.Touch(m.Id.GetNode())

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_ADDRESS_REQUEST:
				node_id, err := nm.Register(m.Data)
				if err != nil {
					clog.Warning("NOCAN_SYS_ADDRESS_REQUEST: Failed to register %s, %s", UdidToString(m.Data), err.Error())
				} else {
					clog.Info("NOCAN_SYS_ADDRESS_REQUEST: Registered %s as node %d", UdidToString(m.Data), node_id)
				}
				msg := NewSystemMessage(0, NOCAN_SYS_ADDRESS_CONFIGURE, uint8(node_id), m.Data)
				nm.Port.SendMessage(msg)
			case NOCAN_SYS_ADDRESS_CONFIGURE_ACK:
				// TODO
			case NOCAN_SYS_ADDRESS_LOOKUP:
				node_id, _ := nm.Lookup(m.Data)
				msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_ADDRESS_LOOKUP_ACK, uint8(node_id), m.Data)
				nm.Port.SendMessage(msg)
			case NOCAN_SYS_CHANNEL_SUBSCRIBE:
				channel_id := BytesToChannel(m.Data)
				if nm.Subscribe(m.Id.GetNode(), channel_id) {
					clog.Info("NOCAN_SYS_CHANNEL_SUBSCRIBE: Node %d successfully subscribed to %d", m.Id.GetNode(), channel_id)
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_SUBSCRIBE: Node %d failed to subscribe to %d", m.Id.GetNode(), channel_id)
				}
			case NOCAN_SYS_CHANNEL_UNSUBSCRIBE:
				channel_id := BytesToChannel(m.Data)
				if nm.Unsubscribe(m.Id.GetNode(), channel_id) {
					clog.Info("NOCAN_SYS_CHANNEL_UNSUBSCRIBE: Node %d successfully unsubscribed to %d", m.Id.GetNode(), channel_id)
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_UNSUBSCRIBE: Node %d failed to unsubscribe to %d", m.Id.GetNode(), channel_id)
				}
			}
		}
	}
}

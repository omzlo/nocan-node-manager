package model

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"pannetrat.com/nocan/bitmap"
	"pannetrat.com/nocan/clog"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

type Node int8

type NodeAttributes map[string]interface{}

type NodeState struct {
	Active        bool           `json:"-"`
	Id            Node           `json:"id"`
	Udid          string         `json:"udid"`
	LastSeen      time.Time      `json:"last_seen"`
	Subscriptions [8]byte        `json:"-"`
	Attributes    NodeAttributes `json:"attributes"`
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
	Mutex    sync.RWMutex
	States   [128]*NodeState
	Udids    map[string]Node
	NodeFile string
}

func NewNodeModel() *NodeModel {
	return &NodeModel{Udids: make(map[string]Node)}
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
			nm.States[v.Node] = &NodeState{Active: false, Id: v.Node, Udid: k, Attributes: v.Attributes}
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

	for i := 0; i < 128; i++ {
		if nm.States[i] == nil {
			nm.States[i] = &NodeState{Active: true, Udid: udid, Id: Node(i)}
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

func (nm *NodeModel) Subscribe(node Node, topic_bitmap []byte) bool {
	if len(topic_bitmap) != 8 {
		return false
	}

	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if ns := nm.getState(node); ns != nil {
		bitmap.Bitmap64Add(ns.Subscriptions[:], topic_bitmap)
		return true
	}
	return false
}

func (nm *NodeModel) Unsubscribe(node Node, topic_bitmap []byte) bool {
	if len(topic_bitmap) != 8 {
		return false
	}

	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if ns := nm.getState(node); ns != nil {
		bitmap.Bitmap64Sub(ns.Subscriptions[:], topic_bitmap)
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

func (nm *NodeModel) Each(fn func(Node, *NodeState, interface{}), data interface{}) {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	for i := 0; i < 128; i++ {
		if ns := nm.getState(Node(i)); ns != nil {
			fn(Node(i), ns, data)
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

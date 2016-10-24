package nocan

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Node int8

type NodeState struct {
	Uid       [8]byte
	LastSeen time.Time
}

func UidToString(id []byte) string {
	retval := ""

	for i := 0; i < len(id); i++ {
		if i > 0 {
			retval += ":"
		}
		retval += hex.EncodeToString(id[i : i+1])
	}
	return retval
}

func StringToUid(s string, id []byte) error {
    src := []byte(s)

    if len(id)<8 {
        return errors.New("Insufficient space to store node uid")
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
	Mutex  sync.Mutex
	States [128]*NodeState
	Uids    map[[8]byte]Node
}

func NewNodeModel() *NodeModel {
	return &NodeModel{Uids: make(map[[8]byte]Node)}
}

func (nm *NodeModel) Register(node [8]byte) (Node, error) {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if n, ok := nm.Uids[node]; ok {
		return n, nil
	}

	for i := 0; i < 128; i++ {
		if nm.States[i] == nil {
			nm.Uids[node] = Node(i)
			nm.States[i] = &NodeState{Uid: node, LastSeen: time.Now()}
			return Node(i), nil
		}
	}
	return Node(-1), errors.New("Maximum number of nodes has been reached.")
}

func (nm *NodeModel) Unregister(node Node) bool {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	ns := nm.getState(node)
	if ns == nil {
		return false
	}
	delete(nm.Uids, ns.Uid)
	nm.States[node] = nil
	return true
}

func (nm *NodeModel) GetProperties(node Node) map[string]interface{} {
    nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if ns := nm.getState(node); ns != nil {
        props := make(map[string]interface{})

        props["id"] = UidToString(ns.Uid[:])
        props["last_seen"] = ns.LastSeen.UTC().String()
        props["attributes"] = make([]string,0)
		return props
	}
    return nil
}

func (nm *NodeModel) ByUid(uid [8]byte) (Node, bool) {
    nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

    if n, ok := nm.Uids[uid]; ok {
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
	return nm.States[node]
}

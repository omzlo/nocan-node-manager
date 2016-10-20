package nocan

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Node int8

type NodeState struct {
	Id       [8]byte
	LastSeen time.Time
}

func IdToString(id []byte) string {
	retval := ""

	for i := 0; i < len(id); i++ {
		if i > 0 {
			retval += ":"
		}
		retval += hex.EncodeToString(id[i : i+1])
	}
	return retval
}

func StringToId(s string) ([]byte, error) {
	res := make([]byte, (len(s)+1)/3)
    src := []byte(s)

	for i := 0; i < len(s); i += 3 {
		if _, err := hex.Decode(res[i/3:i/3+1], src[i:i+2]); err != nil {
			return nil, err
		}
		if i > 0 && src[i-1] != ':' {
			return nil, errors.New("expected ':' in hex identifier")
		}
	}
	return res, nil
}

type NodeModel struct {
	Mutex  sync.Mutex
	States [128]*NodeState
	Ids    map[[8]byte]Node
}

func NewNodeModel() *NodeModel {
	return &NodeModel{Ids: make(map[[8]byte]Node)}
}

func (nm *NodeModel) Register(node [8]byte) (Node, error) {
	nm.Mutex.Lock()
	defer nm.Mutex.Unlock()

	if n, ok := nm.Ids[node]; ok {
		return n, nil
	}

	for i := 0; i < 128; i++ {
		if nm.States[i] == nil {
			nm.Ids[node] = Node(i)
			nm.States[i] = &NodeState{Id: node}
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
	delete(nm.Ids, ns.Id)
	nm.States[node] = nil
	return true
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

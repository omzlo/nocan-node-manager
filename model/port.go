package model

import (
	"pannetrat.com/nocan/clog"
	"sync"
	"time"
)

const (
	DEFAULT_TIMEOUT  = 3 * time.Second
	EXTENDED_TIMEOUT = 12 * time.Second
)

type PortId int

type Port struct {
	Id      PortId
	Name    string
	Manager *PortManager
	Input   chan *Message
	Next    *Port
}

func newPort(manager *PortManager, name string) *Port {
	return &Port{
		Name:    name,
		Manager: manager,
		Input:   make(chan *Message, 4),
	}
}

func (port *Port) SendMessage(m *Message) {
	// Just to make sure a task is not deleted while we iterate through them
	port.Manager.Mutex.Lock()
	defer port.Manager.Mutex.Unlock()

	m.Tag(port.Id)
	//clog.Debug("> Got message on port %d (%s)", port.Id, port.Name)
	for p := port.Manager.Head; p != nil; p = p.Next {
		if p.Id != port.Id { // we could directly compare (p != port), same result.
			//clog.Debug(">> Dispatch message to port %d (%s)", p.Id, p.Name)
			p.Input <- m
		} else {
			//clog.Debug(">> Avoid message to port %d (%s)", p.Id, p.Name)
		}
	}
	//clog.Debug("> Done message on port %d (%s)", port.Id, port.Name)
}

func (port *Port) SendSystemMessage(node Node, fn uint8, param uint8, value []byte) {
	port.SendMessage(NewSystemMessage(node, fn, param, value))
}

func (port *Port) WaitForSystemMessage(node Node, fn uint8, timeout time.Duration) *Message {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case m := <-port.Input:
			clog.Debug("$$$$$ Check that %d == %d and %d == %d", m.Id.GetNode(), node, m.Id.GetSysFunc(), fn)
			if m.Id.GetNode() == node && m.Id.GetSysFunc() == fn {
				return m
			}
		case <-ticker.C: // timeout
			return nil
		}
	}
}

func (port *Port) Publish(node Node, topic Topic, data []byte) {
	m := NewPublishMessage(node, topic, data)
	clog.Debug("Publish %s", m.String())
	port.SendMessage(m)
}

type PortManager struct {
	Mutex     sync.Mutex
	LastId    PortId
	PortCount uint
	Head      *Port
}

func NewPortManager() *PortManager {
	return &PortManager{}
}

/*
func (pm *PortModel) Each(fn func(Port, *PortState, interface{}), extra interface{}) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for iport, vport := range pm.Ports {
		fn(Port(iport), vport, extra)
	}
}
*/

func (pm *PortManager) CreatePort(name string) *Port {
	port := newPort(pm, name)

	pm.Mutex.Lock()

	pm.LastId++
	port.Id = pm.LastId
	port.Next = pm.Head
	pm.Head = port
	pm.PortCount++

	pm.Mutex.Unlock()

	clog.Debug("Created port %d \"%s\"", port.Id, port.Name)

	return port
}

func (pm *PortManager) DestroyPort(port *Port) bool {
	var iter **Port

	clog.Debug("Destroying Port %d \"%s\" - there are %d remaining tasks", port.Id, port.Name, pm.PortCount-1)

	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	iter = &pm.Head

	for *iter != nil {
		if (*iter) == port {
			close((*iter).Input)
			*iter = (*iter).Next
			pm.PortCount--
			return true
		}
		iter = &((*iter).Next)
	}
	return false
}

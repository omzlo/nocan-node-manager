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
	Manager *PortManagerModel
	Input   chan *Message
	Next    *Port
}

func newPort(manager *PortManagerModel, name string) *Port {
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

	//clog.Debug("Send from port %s: %s", port.Name, m.String())
	m.Tag(port.Id)
	for p := port.Manager.Head; p != nil; p = p.Next {
		if p.Id != port.Id { // we could directly compare (p != port), same result.
			//clog.Debug("Send to port %s: %s", p.Name, m.String())
			p.Input <- m
		}
	}
}

func (port *Port) WaitForMessage(checker MessageFilter, timeout time.Duration) *Message {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case m := <-port.Input:
			if checker(m) {
				return m
			}
		case <-ticker.C: // timeout
			return nil
		}
	}
}

type PortManagerModel struct {
	Mutex     sync.Mutex
	LastId    PortId
	PortCount uint
	Head      *Port
}

func NewPortManagerModel() *PortManagerModel {
	return &PortManagerModel{}
}

func (pm *PortManagerModel) CreatePort(name string) *Port {
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

func (pm *PortManagerModel) DestroyPort(port *Port) bool {
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

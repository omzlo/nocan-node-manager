package nocan

import (
	"pannetrat.com/nocan/clog"
	"sync"
)

type Port int

type PortEndpoint interface {
	GetType() string
	GetAttributes() interface{}
	ProcessSend(*PortModel, Port)
	ProcessRecv(*PortModel, Port)
}

type PortState struct {
	Endpoint PortEndpoint
	Outputs  chan *Message
	Signals  chan Signal
}

func NewPortState(e PortEndpoint) *PortState {
	return &PortState{Endpoint: e, Outputs: make(chan *Message, 4), Signals: make(chan Signal, 4)}
}

type PortModel struct {
	Mutex sync.Mutex
	//LastPort Port
	//	Input        chan *Message
	//	InputSignals chan Signal
	Ports []*PortState
}

func NewPortModel() *PortModel {
	return &PortModel{Ports: make([]*PortState, 0, 16)}
}

func (pm *PortModel) Each(fn func(Port, *PortState, interface{}), extra interface{}) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	for iport, vport := range pm.Ports {
		fn(Port(iport), vport, extra)
	}
}

func (pm *PortModel) Add(e PortEndpoint) Port {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	retval := len(pm.Ports)
	pm.Ports = append(pm.Ports, NewPortState(e))
	return Port(retval)
}

var messageCount uint = 0

func (pm *PortModel) SendMessage(srcPort Port, m *Message) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	m.Port = srcPort
	messageCount++
	// clog.Debug("Sending message %d from port %d to all other ports: %s", messageCount, srcPort, m.String())
	for cindex, cstate := range pm.Ports {
		if Port(cindex) != m.Port {
			// clog.Debug("Dispatching message %d to port %d", messageCount, cindex)
			cstate.Outputs <- m
		}
	}
}

func (pm *PortModel) SendSignal(port Port, value uint) {
	pm.Mutex.Lock()
	defer pm.Mutex.Unlock()

	// clog.Debug("Raised signal 0x%08x on port %d for all other ports", value, port)
	for cindex, cstate := range pm.Ports {
		if Port(cindex) != port {
			// clog.Debug("Dispatching signal 0x%08x to port %d", value, cindex)
			cstate.Signals <- CreateSignal(port, value)
		}
	}
}

func (pm *PortModel) Recv(port Port) (*Message, Signal) {
	// No locking needed here because of channels

	select {
	case m := <-pm.Ports[port].Outputs:
		// clog.Debug("Got message from chan port %d tagged with port %d", port, m.Port)
		return m, CreateSignal(0, 0)
	case s := <-pm.Ports[port].Signals:
		// clog.Debug("Got signal on port %d with value 0x%08x, originating from port %d", port, s.Value, s.Port)
		return nil, s
	}
}

func (pm *PortModel) ListenAndServe() {
	clog.Debug("There are %d running ports", len(pm.Ports))
	for cindex, cstate := range pm.Ports {
		go cstate.Endpoint.ProcessRecv(pm, Port(cindex))
		go cstate.Endpoint.ProcessSend(pm, Port(cindex))
		clog.Debug("Port %d: Output = chan%v, type=%s", cindex, cstate.Outputs, cstate.Endpoint.GetType())
	}
	/*
		for {
			select {
			case m := <-pm.Input:
				clog.Debug("Got a message on chan %v tagged with port %d.", pm.Input, int(m.Port))
				for cindex, cstate := range pm.Ports {
					if Port(cindex) != m.Port {
						clog.Debug("Dispatching to channel %v.", cstate.Output)
						cstate.Output <- m
					}
				}
				//case s := <-pm.InputSignal:
				//clog.Debug("Dispatching signal
				// add signal port ? for errors and timers ?
			}
		}
	*/
}

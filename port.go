package nocan

import (
	"pannetrat.com/nocan/clog"
)

type Signal struct {
	Port
	Value uint
}

type Port int

type PortEndpoint interface {
	ProcessSend(*PortModel, Port)
	ProcessRecv(*PortModel, Port)
}

type PortState struct {
	Endpoint      PortEndpoint
	Output        chan *Message
	OutputSignals chan Signal
}

func NewPortState(e PortEndpoint) *PortState {
	return &PortState{Endpoint: e, Output: make(chan *Message, 4)}
}

type PortModel struct {
	//LastPort Port
	Input        chan *Message
	InputSignals chan Signal
	Ports        []*PortState
}

func NewPortModel() *PortModel {
	return &PortModel{Signals: make(chan Signal), Input: make(chan *Message, 4), Ports: make([]*PortState, 0, 16)}
}

func (pm *PortModel) Add(e PortEndpoint) Port {
	retval := len(pm.Ports)
	pm.Ports = append(pm.Ports, NewPortState(e))
	return Port(retval)
}

func (pm *PortModel) Send(port Port, m *Message) {
	m.Port = port
	pm.Input <- m
	clog.Debug("Added message to chan %v tagged with port %d", pm.Input, port)
}

func (pm *PortModel) SendSignal(port Port, signal uint) {
	pm.InputSignal <- Signal{port, signal}
	clog.Debug("Raised signal %08x on port %d", signal, port)
}

func (pm *PortModel) Recv(port Port) (*Message, Signal) {
	select {
	case m := <-pm.Ports[port].Output:
		clog.Debug("Got message from chan %v (port %d) tagged with port %d", pm.Ports[port].Output, port, m.Port)
		return m, 0
	case s := <-pm.Ports[port].Signals:
		clog.Debuf("Got signal on port %d with value 0x%8x", port, s)
		return nil, s
	}
}

func (pm *PortModel) ListenAndServe() {
	clog.Debug("There are %d running ports", len(pm.Ports))
	for cindex, cstate := range pm.Ports {
		go cstate.Endpoint.ProcessRecv(pm, Port(cindex))
		go cstate.Endpoint.ProcessSend(pm, Port(cindex))
		clog.Debug("Port %d: Input chan=%v Output= chan%v", cindex, pm.Input, cstate.Output)
	}
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
		case s := <-pm.InputSignal:
			clog.Debugf
			// add signal port ? for errors and timers ?
		}
	}
}

type LogEndpoint struct {
}

func (ld *LogEndpoint) ProcessSend(pm *PortModel, p Port) {
	return // nothing to do
}

func (ld *LogEndpoint) ProcessRecv(pm *PortModel, p Port) {
	for {
		m := pm.Recv(p)
		clog.Debug("Message %s", m.String())
	}
}

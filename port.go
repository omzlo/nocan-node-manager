package nocan

type Port int

type PortEndpoint interface {
	ProcessSend(*PortModel, Port)
	ProcessRecv(*PortModel, Port)
}

type PortState struct {
	Endpoint PortEndpoint
	Output   chan *Message
}

func NewPortState(e PortEndpoint) *PortState {
	return &PortState{Endpoint: e, Output: make(chan *Message, 4)}
}

type PortModel struct {
	//LastPort Port
	Input chan *Message
	Ports []*PortState
}

func NewPortModel() *PortModel {
	return &PortModel{Input: make(chan *Message, 4), Ports: make([]*PortState, 0, 16)}
}

func (pm *PortModel) Add(e PortEndpoint) Port {
	retval := len(pm.Ports)
	pm.Ports = append(pm.Ports, NewPortState(e))
	return Port(retval)
}

func (pm *PortModel) Send(port Port, m *Message) {
	m.Port = port
	pm.Input <- m
	Log(DEBUG, "Added message to chan %v tagged with port %d", pm.Input, port)
}

func (pm *PortModel) Recv(port Port) *Message {
	m := <-pm.Ports[port].Output
	Log(DEBUG, "Got message from chan %v tagged with port %d", pm.Ports[port].Output, m.Port)
	return m
}

func (pm *PortModel) ListenAndServe() {
	Log(DEBUG, "There are %d running ports", len(pm.Ports))
	for cindex, cstate := range pm.Ports {
		go cstate.Endpoint.ProcessRecv(pm, Port(cindex))
		go cstate.Endpoint.ProcessSend(pm, Port(cindex))
		Log(DEBUG, "Port %d: Input chan=%v Output= chan%v", cindex, pm.Input, cstate.Output)
	}
	for {
		select {
		case m := <-pm.Input:
			Log(DEBUG, "Got a message on chan %v tagged with port %d.", pm.Input, int(m.Port))
			for cindex, cstate := range pm.Ports {
				if Port(cindex) != m.Port {
					Log(DEBUG, "Dispatching to channel %v.", cstate.Output)
					cstate.Output <- m
				}
			}
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
		Log(DEBUG, "Message %s", m.String())
	}
}

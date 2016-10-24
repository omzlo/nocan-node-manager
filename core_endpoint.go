package nocan

import (
	"pannetrat.com/nocan/clog"
	"time"
)

type CoreEndpoint struct {
	PortModel *PortModel
	Port
	Topics *TopicController
	Nodes  *NodeController
	ToSend chan *Message
}

func NewCoreEndpoint() *CoreEndpoint {
	endpoint := &CoreEndpoint{}
	endpoint.Topics = NewTopicController(endpoint)
	endpoint.Nodes = NewNodeController(endpoint)
	endpoint.ToSend = make(chan *Message)
	return endpoint
}

func (ce *CoreEndpoint) GetType() string {
	return "core"
}

func (ce *CoreEndpoint) GetAttributes() interface{} {
	return nil
}

func (ce *CoreEndpoint) ProcessSend(pm *PortModel, p Port) {
	/*
		for {
			m := <-ce.ToSend
			clog.Debug("CoreProcessSend %s", m.String())
			pm.SendMessage(p, m)
		}
	*/
	ce.PortModel = pm
	ce.Port = p
	go func() {
		for {
			time.Sleep(10 * time.Second)
			pm.SendSignal(p, SIGNAL_HEARTBEAT)
		}
	}()
}

func (ce *CoreEndpoint) ProcessRecv(pm *PortModel, p Port) {
	for {
		pm.Recv(p)
	}
}

func (ce *CoreEndpoint) Publish(node Node, topic Topic, data []byte) {
	clog.Debug("Publish node=%d, topic=%d dlen=%d", int(node), int(topic), len(data))
	m := NewPublishMessage(node, topic, data)
	clog.Debug("Publish %s", m.String())
	ce.PortModel.SendMessage(ce.Port, m)
}

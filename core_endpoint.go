package nocan

import (
    "pannetrat.com/nocan/clog"
)

type CoreEndpoint struct {
	Topics *TopicController
    Nodes *NodeController
	ToSend chan *Message
}

func NewCoreEndpoint() *CoreEndpoint {
    endpoint := &CoreEndpoint{}
    endpoint.Topics = NewTopicController(endpoint)
    endpoint.Nodes = NewNodeController(endpoint)
    endpoint.ToSend = make(chan *Message)
    return endpoint
}

func (ce *CoreEndpoint) ProcessSend(pm *PortModel, p Port) {
	for {
		m := <-ce.ToSend
        clog.Debug("CoreProcessSend %s", m.String())
		pm.Send(p,m)
	}
}

func (ce *CoreEndpoint) ProcessRecv(pm *PortModel, p Port) {
	for {
		pm.Recv(p)
	}
}

func (ce *CoreEndpoint) Publish(node Node, topic Topic, data []byte) {
    clog.Debug("Publish node=%d, topic=%d dlen=%d",int(node),int(topic),len(data))
    m := NewPublishMessage(node, topic, data)
    clog.Debug("Publish %s",m.String())
	ce.ToSend <- m
}

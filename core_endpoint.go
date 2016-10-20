package nocan

type CoreEndpoint struct {
	Topics *TopicController
	ToSend chan *Message
}

func NewCoreEndpoint() *CoreEndpoint {
    endpoint := &CoreEndpoint{}
    endpoint.Topics = NewTopicController(endpoint)
    endpoint.ToSend = make(chan *Message)
    return endpoint
}

func (ce *CoreEndpoint) ProcessSend(pm *PortModel, p Port) {
	for {
		m := <-ce.ToSend
        Log(DEBUG,"CoreProcessSend %s", m.String())
		pm.Send(p,m)
	}
}

func (ce *CoreEndpoint) ProcessRecv(pm *PortModel, p Port) {
	for {
		pm.Recv(p)
	}
}

func (ce *CoreEndpoint) Publish(node Node, topic Topic, data []byte) {
    Log(DEBUG,"Publish node=%d, topic=%d dlen=%d",int(node),int(topic),len(data))
    m := NewPublishMessage(node, topic, data)
    Log(DEBUG,"Publish %s",m.String())
	ce.ToSend <- m
}

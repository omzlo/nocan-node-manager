package nocan

import (
	"pannetrat.com/nocan/bitmap"
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"time"
)

type CoreEndpoint struct {
	Application *ApplicationController
}

func NewCoreEndpoint(app *ApplicationController) *CoreEndpoint {
	endpoint := &CoreEndpoint{Application: app}
	return endpoint
}

func (ce *CoreEndpoint) GetType() string {
	return "core"
}

func (ce *CoreEndpoint) GetAttributes() interface{} {
	return nil
}

func (ce *CoreEndpoint) ProcessSend(pm *model.PortModel, p model.Port) {
	clock := ce.Application.Topics.Model.FindByName("/clock")
	var dummy [8]byte
	go func() {
		for {
			time.Sleep(10 * time.Second)
			//pm.SendSignal(p, SIGNAL_HEARTBEAT)
			if clock >= 0 {
				ce.Application.Publish(0, clock, dummy[:])
			}
		}
	}()
}

func (ce *CoreEndpoint) ProcessRecv(pm *model.PortModel, p model.Port) {
	for {
		m, _ := pm.Recv(p)
		if m != nil {
			if m.Id.IsSystem() {
				switch m.Id.GetSysFunc() {
				case NOCAN_SYS_ADDRESS_REQUEST:
					node_id, err := ce.Application.Nodes.Model.Register(m.Data)
					if err != nil {
						clog.Warning("NOCAN_SYS_ADDRESS_REQUEST: Failed to register %s, %s", model.UidToString(m.Data), err.Error())
					} else {
						clog.Info("NOCAN_SYS_ADDRESS_REQUEST: Registered %s as node %d", model.UidToString(m.Data), node_id)
					}
					msg := model.NewSystemMessage(0, NOCAN_SYS_ADDRESS_CONFIGURE, uint8(node_id), m.Data)
					ce.Application.SendMessage(msg)
				case NOCAN_SYS_ADDRESS_CONFIGURE_ACK:
					// TODO
				case NOCAN_SYS_ADDRESS_LOOKUP:
					node_id, _ := ce.Application.Nodes.Model.Lookup(m.Data)
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_ADDRESS_LOOKUP_ACK, uint8(node_id), m.Data)
					ce.Application.SendMessage(msg)
				case NOCAN_SYS_TOPIC_REGISTER:
					topic_id, err := ce.Application.Topics.Model.Register(string(m.Data))
					if err != nil {
						clog.Warning("NOCAN_SYS_TOPIC_REGISTER: Failed to register topic %s, %s", string(m.Data), err.Error())
					} else {
						clog.Info("NOCAN_SYS_TOPIC_REGISTER: Registered topic %s as %d", string(m.Data), topic_id)
					}
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_REGISTER_ACK, uint8(topic_id), nil)
					ce.Application.SendMessage(msg)
				case NOCAN_SYS_TOPIC_LOOKUP:
					var bitmap [8]byte
					if ce.Application.Topics.Model.Lookup(string(m.Data), bitmap[:]) {
						clog.Info("NOCAN_SYS_TOPIC_LOOKUP: Node %d succesfully found bitmap for topic %s", m.Id.GetNode(), string(m.Data))
						msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0, bitmap[:])
						ce.Application.SendMessage(msg)
					} else {
						clog.Warning("NOCAN_SYS_TOPIC_LOOKUP: Node %d failed to find bitmap for topic %s", m.Id.GetNode(), string(m.Data))
						msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_LOOKUP_ACK, 0xFF, nil)
						ce.Application.SendMessage(msg)
					}
				case NOCAN_SYS_TOPIC_UNREGISTER:
					var rval uint8
					if ce.Application.Topics.Model.Unregister(model.Topic(m.Id.GetSysParam())) {
						clog.Info("NOCAN_SYS_TOPIC_UNREGISTER: Node %d successfully unregistered topic %d", m.Id.GetNode(), m.Id.GetSysParam())
						rval = 0
					} else {
						clog.Warning("NOCAN_SYS_TOPIC_UNREGISTER: Node %d failed to unregister topic %d", m.Id.GetNode(), m.Id.GetSysParam())
						rval = 0xFF
					}
					msg := model.NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_TOPIC_UNREGISTER_ACK, rval, nil)
					ce.Application.SendMessage(msg)
				case NOCAN_SYS_TOPIC_SUBSCRIBE:
					if ce.Application.Nodes.Model.Subscribe(m.Id.GetNode(), m.Data) {
						clog.Info("NOCAN_SYS_TOPIC_SUBSCRIBE: Node %d successfully subscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
					} else {
						clog.Warning("NOCAN_SYS_TOPIC_SUBSCRIBE: Node %d failed to subscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
					}
				case NOCAN_SYS_TOPIC_UNSUBSCRIBE:
					if ce.Application.Nodes.Model.Unsubscribe(m.Id.GetNode(), m.Data) {
						clog.Info("NOCAN_SYS_TOPIC_UNSUBSCRIBE: Node %d successfully unsubscribed to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
					} else {
						clog.Warning("NOCAN_SYS_TOPIC_UNSUBSCRIBE: Node %d failed to unsubscribe to %v", m.Id.GetNode(), bitmap.Bitmap64ToSlice(m.Data))
					}
				case NOCAN_SYS_NODE_BOOT_ACK:
					// TODO:
				default:
					clog.Warning("Got unknown SYS message func %d", m.Id.GetSysFunc())
				}
			}
		}
	}
}

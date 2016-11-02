package nocan

import (
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
)

type LogEndpoint struct {
}

func NewLogEndpoint(app *ApplicationController) *LogEndpoint {
	endpoint := &LogEndpoint{}
	return endpoint
}

func (ld *LogEndpoint) GetType() string {
	return "log"
}

func (ld *LogEndpoint) GetAttributes() interface{} {
	return nil
}

func (ld *LogEndpoint) ProcessSend(pm *model.PortModel, p model.Port) {
	return // nothing to do
}

func (ld *LogEndpoint) ProcessRecv(pm *model.PortModel, p model.Port) {
	for {
		m, s := pm.Recv(p)
		if m != nil {
			clog.Info("LOG PORT: Message %s", m.String())
		} else {
			clog.Info("LOG PORT: Signal 0x%08x", s.Value)
		}
	}
}

package nocan

import (
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/controller"
	"pannetrat.com/nocan/model"
)

type LogTask struct {
	Port *model.Port
}

func NewLogTask(app *controller.Application) *LogTask {
	task := &LogTask{Port: model.PortManager.CreatePort("log")}
	return task
}

func (lt *LogTask) Run() {
	for {
		m := <-lt.Port.Input
		clog.Info("LOG PORT: Message %s", m.String())
	}
}

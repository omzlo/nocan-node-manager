package nocan

import (
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
)

type LogTask struct {
	Port *model.Port
}

func NewLogTask(pm *model.PortManager) *LogTask {
	task := &LogTask{}
	task.Port = pm.CreatePort("log")
	return task
}

func (lt *LogTask) Run() {
	for {
		m := <-lt.Port.Input
		clog.Info("LOG PORT: Message %s", m.String())
	}
}

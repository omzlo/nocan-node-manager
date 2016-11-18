package nocan

import (
	//"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
)

type BaseTask struct {
	PortManager *model.PortManager
	Port        *model.Port
}

func (task *BaseTask) Run() {
	if task.Port == nil {
		panic("Port is uninitialized in Task.Run()")
	}
	//clog.Debug("+ Task run started on on port %d (%s)", task.Port.Id, task.Port.Name)
	for {
		//clog.Debug("++ Task waiting for message on port %d (%s)", task.Port.Id, task.Port.Name)
		<-task.Port.Input
		//clog.Debug("++ Task got message with src_port=%d on port %d (%s)", m.SourcePort, task.Port.Id, task.Port.Name)
	}
}

package nocan

import (
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/controllers"
	"pannetrat.com/nocan/models"
)

type LogTask struct {
	Port *models.Port
}

func NewLogTask(app *controllers.Application) *LogTask {
	task := &LogTask{Port: models.PortManager.CreatePort("log")}
	return task
}

func (lt *LogTask) Run() {
	for {
		m := <-lt.Port.Input
		clog.Info("LOG PORT: Message %s", m.String())
	}
}

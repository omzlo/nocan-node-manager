package nocan

import (
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
)

type LogTask struct {
}

func NewLogTask() *LogTask {
	return &LogTask{}
}

func (lt *LogTask) Setup(_ *model.TaskState) {

}

func (lt *LogTask) Run(task *model.TaskState) {
	for {
		m, s := task.Recv()
		if m != nil {
			clog.Info("LOG PORT: Message %s", m.String())
		} else {
			clog.Info("LOG PORT: Signal 0x%08x", s.Value)
		}
	}
}

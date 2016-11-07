package model

type Signal struct {
	Task
	Value uint
}

const (
	SIGNAL_HEARTBEAT = 1
)

func CreateSignal(task Task, value uint) Signal {
	return Signal{task, value}
}

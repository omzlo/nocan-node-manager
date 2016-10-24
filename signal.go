package nocan

type Signal struct {
	Port
	Value uint
}

const (
	SIGNAL_HEARTBEAT = 1
)

func CreateSignal(port Port, value uint) Signal {
	return Signal{port, value}
}

package model

import (
	"errors"
	"sync"
)

const (
	DRIVER_POWER_OFF = 0
	DRIVER_POWER_ON  = 1
)

var (
	DriverBusyError         = errors.New("Driver is busy, try again later")
	DriverDisconnectedError = errors.New("Driver is disconnected")
)

type DriverId uint8

type Driver struct {
	BaseTask
	DriverId
	Access       sync.Mutex
	Serial       *serialcan.SerialCan
	DeviceName   string
	InputBuffer  [128]*model.Message
	OutputBuffer model.Message
	//Ticker        *time.Ticker
	//RequestStatus [6]SerialCanRequest
	PowerStatus struct {
		PowerOn      bool
		PowerLevel   float32
		SenseOn      bool
		SenseLevel   float32
		UsbReference float32
	}
	Connected bool
}

func Open(pm deviceName string) (*Driver, error) {
	serial, err := serialcan.SerialCanOpen(device)
	if err != nil {
		clog.Error("Could not open %s: %s", device, err.Error())
		return nil, err
	}
	clog.Debug("Opened device %s", device)

	driver := &Driver{BaseTask: BaseTask{pm, pm.CreatePort("serial")}, Serial: serial, Device: device}

	return se

	return nil, nil
}

func (ds *Driver) Rescue() bool {
	return false
}

func (ds *Driver) Close() {
	return
}

func (ds *Driver) PerformSetPower(power uint8) error {
	ds.Access.Lock()
	defer ds.Access.Unlock()

	if !ds.Connected {
		return DriverDisconnectedError
	}
	//
	return nil
}

func (ds *Driver) PerformUpdatePowerStatus() error {
	ds.Access.Lock()
	defer ds.Access.Unlock()

	if !ds.Connected {
		return DriverDisconnectedError
	}
	//
	return nil
}

func (ds *Driver) PerformReset() error {
	ds.Access.Lock()
	defer ds.Access.Unlock()

	if !ds.Connected {
		return DriverDisconnectedError
	}
	//
	return nil
}

type DriverModel struct {
	Drivers []*Driver
}

func NewDriverModel() *DriverModel {
	return &DriverModel{Drivers: make([]*Driver, 2)}
}

func (dm *DriverModel) Add(dr *Driver) {
	dm.Drivers = append(dm.Drivers, dr)
}

func (dm *DriverModel) GetDriver(id DriverId) *Driver {
	if int(id) >= len(dm.Drivers) {
		return nil
	}
	return dm.Drivers[id]
}

/*

for the controller:

GET /api/driver/:id
POST /api/driver/:id/control?c=reset
POST /api/driver/:id/control?c=poweron
POST /api/driver/:id/control?c=poweroff
*/

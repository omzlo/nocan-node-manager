package model

import (
	"errors"
	"pannetrat.com/nocan/clog"
	"sync"
	"time"
)

const (
	DRIVER_POWER_OFF = 0
	DRIVER_POWER_ON  = 1
)

var (
	DriverBusyError         = errors.New("Driver is busy, try again later")
	DriverDisconnectedError = errors.New("Driver is disconnected")
	DriverInputError        = errors.New("Driver input error")
	DriverTimeoutError      = errors.New("Operation timed out")
	DriverOperationError    = errors.New("Operation failed")
)

type SerialCanRequest struct {
	Submitted bool
	C         chan uint8
}

const (
	SERIAL_CAN_REQUEST_STATUS_SUCCESS = 1
	SERIAL_CAN_REQUEST_STATUS_FAILURE = 2
	SERIAL_CAN_REQUEST_STATUS_TIMEOUT = 3
)

const (
	SERIAL_CAN_CTRL_RESET              = 1
	SERIAL_CAN_CTRL_GET_REGISTERS      = 2
	SERIAL_CAN_CTRL_SET_POWER          = 3
	SERIAL_CAN_CTRL_GET_POWER_STATUS   = 4
	SERIAL_CAN_CTRL_RESISTOR_CONFIGURE = 5
)

type DriverId uint8

type Driver struct {
	DriverId      `json:"id"`
	Access        sync.Mutex          `json:"-"`
	Serial        *SerialCan          `json:"-"`
	DeviceName    string              `json:"device_name"`
	InputBuffer   [128]*Message       `json:"-"`
	OutputBuffer  Message             `json:"-"`
	RequestStatus [6]SerialCanRequest `json:"-"`
	PowerStatus   struct {
		PowerOn      bool    `json:"power_on"`
		PowerLevel   float32 `json:"power_level"`
		SenseOn      bool    `json:"sense_on"`
		SenseLevel   float32 `json:"sense_level"`
		UsbReference float32 `json:"usb_reference"`
	} `json:"power_status"`
	Connected bool `json:"connected"`
}

func NewDriver(deviceName string) (*Driver, error) {
	serial, err := SerialCanOpen(deviceName)
	if err != nil {
		clog.Error("Could not open %s: %s", deviceName, err.Error())
		return nil, err
	}
	//clog.Debug("Opened device %s", deviceName)

	driver := &Driver{Serial: serial, DeviceName: deviceName, Connected: true}

	return driver, nil
}

func (ds *Driver) Rescue() bool {
	ds.Close()
	for {
		serial, err := SerialCanOpen(ds.DeviceName)
		if err == nil {
			ds.Serial = serial
			clog.Info("Reopened device %s", ds.DeviceName)
			ds.SendReset()
			return true
		} else {
			clog.Warning("Failed to reopen device %s: %s", ds.DeviceName, err.Error())
		}
		time.Sleep(10 * time.Second)
	}
}

func (ds *Driver) Close() {
	ds.Serial.Close()
}

func (ds *Driver) finalizeAction(action uint8, result uint8) {
	ds.RequestStatus[action].C <- result

	ds.Access.Lock()
	ds.RequestStatus[action].Submitted = false
	ds.Access.Unlock()
}

func (ds *Driver) performAction(action uint8, param uint8, data []byte) (*SerialCanRequest, error) {
	ds.Access.Lock()
	defer ds.Access.Unlock()

	if !ds.Connected {
		return nil, DriverDisconnectedError
	}

	if ds.RequestStatus[action].Submitted {
		return nil, DriverBusyError
	}

	ds.RequestStatus[action].Submitted = true
	ds.RequestStatus[action].C = make(chan uint8, 1)

	go func() {
		time.Sleep(3 * time.Second)
		ds.RequestStatus[action].C <- SERIAL_CAN_REQUEST_STATUS_TIMEOUT
		ds.Access.Lock()
		ds.RequestStatus[action].Submitted = false
		ds.Access.Unlock()
	}()

	var frame CanFrame
	frame.CanId = CanId(CANID_MASK_CONTROL).SetSysFunc(action).SetSysParam(param)
	frame.CanDlc = uint8(len(data))
	if data != nil {
		copy(frame.CanData[:], data)
	}
	ds.Serial.Send(&frame)
	return &ds.RequestStatus[action], nil
}

func (ds *Driver) SendSetPower(power uint8) (*SerialCanRequest, error) {
	return ds.performAction(SERIAL_CAN_CTRL_SET_POWER, power, nil)
}

func (ds *Driver) SendUpdatePowerStatus() (*SerialCanRequest, error) {
	return ds.performAction(SERIAL_CAN_CTRL_GET_POWER_STATUS, 0, nil)
}

func (ds *Driver) SendReset() (*SerialCanRequest, error) {
	return ds.performAction(SERIAL_CAN_CTRL_RESET, 0, nil)
}

func (ds *Driver) ProcessFrames(port *Port) {
	for {
		var frame CanFrame

		//clog.Debug("Waiting for serial input")
		if ds.Serial.Recv(&frame) == false {
			clog.Error("Failed to receive frame from %s", ds.DeviceName)
			if ds.Rescue() {
				continue
			} else {
				break
			}
		}
		//clog.Debug("Got serial input")

		node := frame.CanId.GetNode()

		switch {
		case !frame.CanId.IsExtended(), frame.CanId.IsRemote():
			clog.Warning("Got malformed frame, discarding.")
		case frame.CanId.IsControl():
			switch frame.CanId.GetSysFunc() {
			case SERIAL_CAN_CTRL_RESET:
				if frame.CanId.GetSysParam() == 0 {
					ds.finalizeAction(SERIAL_CAN_CTRL_RESET, SERIAL_CAN_REQUEST_STATUS_SUCCESS)
				} else {
					ds.finalizeAction(SERIAL_CAN_CTRL_RESET, SERIAL_CAN_REQUEST_STATUS_FAILURE)
				}
			case SERIAL_CAN_CTRL_SET_POWER:
				if frame.CanId.GetSysParam() == 0xAA || frame.CanId.GetSysParam() == 0xDD {
					ds.finalizeAction(SERIAL_CAN_CTRL_SET_POWER, SERIAL_CAN_REQUEST_STATUS_SUCCESS)
				} else {
					ds.finalizeAction(SERIAL_CAN_CTRL_SET_POWER, SERIAL_CAN_REQUEST_STATUS_FAILURE)
				}
			case SERIAL_CAN_CTRL_GET_POWER_STATUS:
				if frame.CanId.GetSysParam() == 0 {
					usbref := (uint16(frame.CanData[6]) << 8) | uint16(frame.CanData[7])
					powerlevel := (uint16(frame.CanData[1]) << 8) | uint16(frame.CanData[2])
					senselevel := (uint16(frame.CanData[4]) << 8) | uint16(frame.CanData[5])
					ds.PowerStatus.PowerOn = frame.CanData[0] != 0
					ds.PowerStatus.PowerLevel = float32(powerlevel) / float32(usbref) * 1.1 * 9.2
					ds.PowerStatus.SenseOn = frame.CanData[3] != 0
					ds.PowerStatus.SenseLevel = 100 * float32(senselevel) / 1023
					ds.PowerStatus.UsbReference = 1023 * 1.1 / float32(usbref)
					clog.Info("Power stat estimates: power=%t power_level=%.2fV sense=%t sense_level=%.3f%% usb_power=%.2fV",
						ds.PowerStatus.PowerOn, ds.PowerStatus.PowerLevel,
						ds.PowerStatus.SenseOn, ds.PowerStatus.SenseLevel,
						ds.PowerStatus.UsbReference)
					ds.finalizeAction(SERIAL_CAN_CTRL_GET_POWER_STATUS, SERIAL_CAN_REQUEST_STATUS_SUCCESS)
				} else {
					ds.finalizeAction(SERIAL_CAN_CTRL_GET_POWER_STATUS, SERIAL_CAN_REQUEST_STATUS_FAILURE)
				}
			}
		case frame.CanId.IsError():
			clog.Error("Recieved error frame on CAN controller")
		default:
			if frame.CanId.IsFirst() {
				if ds.InputBuffer[node] != nil {
					clog.Warning("Got frame with inconsistent first bit indicator, discarding.")
					break
				}
				ds.InputBuffer[node] = NewMessageFromFrame(&frame)
			} else {
				if ds.InputBuffer[node] == nil {
					clog.Warning("Got first frame with missing first bit indicator, discarding.")
					break
				}
				ds.InputBuffer[node].AppendData(frame.CanData[:frame.CanDlc])
			}
			if frame.CanId.IsLast() {
				port.SendMessage(ds.InputBuffer[node])
				ds.InputBuffer[node] = nil
			}
		}
	}
}

func (ds *Driver) ProcessMessages(port *Port) {
	ticker := time.NewTicker(10 * time.Second)

	for {
		var frame CanFrame

		select {
		case m := <-port.Input:
			pos := 0
			for {
				frame.CanId = (m.Id & CANID_MASK_MESSAGE) | CANID_MASK_EXTENDED
				if pos == 0 {
					frame.CanId |= CANID_MASK_FIRST
				}
				if len(m.Data)-pos <= 8 {
					frame.CanId |= CANID_MASK_LAST
					frame.CanDlc = uint8(len(m.Data) - pos)
				} else {
					frame.CanDlc = 8
				}
				copy(frame.CanData[:], m.Data[pos:pos+int(frame.CanDlc)])
				if ds.Serial.Send(&frame) == false {
					clog.Error("Failed to send frame to %s", ds.DeviceName)
					return
				}
				pos += int(frame.CanDlc)
				if pos >= len(m.Data) {
					break
				}
			}
		case <-ticker.C:
			ds.SendUpdatePowerStatus()
		}
	}
}

func (ds *Driver) Run(port *Port) {
	go ds.ProcessFrames(port)
	ds.SendReset()
	ds.SendUpdatePowerStatus()
	ds.ProcessMessages(port)
}

type DriverModel struct {
	Drivers []*Driver
}

func NewDriverModel() *DriverModel {
	return &DriverModel{Drivers: make([]*Driver, 0, 2)}
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

func (dm *DriverModel) Run(port *Port) {
	for _, driver := range dm.Drivers {
		go driver.Run(port)
	}
}

/*

for the controller:

GET /api/driver/:id
POST /api/driver/:id/control?c=reset
POST /api/driver/:id/control?c=poweron
POST /api/driver/:id/control?c=poweroff
*/

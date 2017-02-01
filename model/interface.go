package model

import (
	"errors"
	"fmt"
	"pannetrat.com/nocan/clog"
	"sync"
	"time"
)

const (
	INTERFACE_POWER_OFF = 0
	INTERFACE_POWER_ON  = 1
)

var (
	InterfaceBusyError         = errors.New("Interface is busy, try again later")
	InterfaceDisconnectedError = errors.New("Interface is disconnected")
	InterfaceInputError        = errors.New("Interface input error")
	InterfaceTimeoutError      = errors.New("Operation timed out")
	InterfaceOperationError    = errors.New("Operation failed")
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

const (
	POWER_FLAGS_SUPPLY = 1
	POWER_FLAGS_SENSE  = 2
	POWER_FLAGS_FAULT  = 4
)

type InterfaceId uint8

type Interface struct {
	InterfaceId   `json:"id"`
	Access        sync.Mutex          `json:"-"`
	Serial        *SerialCan          `json:"-"`
	DeviceName    string              `json:"device_name"`
	InputBuffer   [128]*Message       `json:"-"`
	OutputBuffer  Message             `json:"-"`
	RequestStatus [6]SerialCanRequest `json:"-"`
	PowerStatus   struct {
		PowerOn      bool    `json:"power_on"`
		SenseOn      bool    `json:"sense_on"`
		Fault        bool    `json:"fault"`
		PowerLevel   float32 `json:"power_level"`
		SenseLevel   float32 `json:"sense_level"`
		UsbReference float32 `json:"usb_reference"`
	} `json:"power_status"`
	Connected bool `json:"connected"`
}

func NewInterface(deviceName string) (*Interface, error) {
	serial, err := SerialCanOpen(deviceName)
	if err != nil {
		clog.Error("Could not open %s: %s", deviceName, err.Error())
		return nil, err
	}
	//clog.Debug("Opened device %s", deviceName)

	driver := &Interface{Serial: serial, DeviceName: deviceName, Connected: true}

	if err = driver.Reset(); err != nil {
		clog.Error(err.Error())
		return nil, err
	}

	return driver, nil
}

func (ds *Interface) Reset() error {
	var frame CanFrame

	frame.CanId = CanId(CANID_MASK_CONTROL).SetSysFunc(SERIAL_CAN_CTRL_RESET).SetSysParam(0)
	frame.CanDlc = 0
	ds.Serial.Send(&frame)

	status := make(chan error, 1)

	go func() {
		time.Sleep(3 * time.Second)
		status <- fmt.Errorf("Timeout while resetting %s", ds.DeviceName)
	}()
	go func() {
		if ds.Serial.Recv(&frame) == false {
			status <- fmt.Errorf("Failed to receive data from %s", ds.DeviceName)
		}
		if !frame.CanId.IsExtended() || frame.CanId.GetSysFunc() != SERIAL_CAN_CTRL_RESET || frame.CanId.GetSysParam() != 0 {
			status <- fmt.Errorf("Incorrect response to reset from %s", ds.DeviceName)
		}
		status <- nil
	}()

	err := <-status
	return err
}

func (ds *Interface) Rescue() bool {
	ds.Close()
	for {
		serial, err := SerialCanOpen(ds.DeviceName)
		if err == nil {
			ds.Serial = serial
			clog.Info("Reopened device %s", ds.DeviceName)
			if err = ds.Reset(); err != nil {
				clog.Warning(err.Error())
				return false
			} else {
				return true
			}
		} else {
			clog.Warning("Failed to reopen device %s: %s", ds.DeviceName, err.Error())
		}
		time.Sleep(10 * time.Second)
	}
}

func (ds *Interface) Close() {
	ds.Serial.Close()
}

func (ds *Interface) finalizeAction(action uint8, result uint8) {
	ds.RequestStatus[action].C <- result

	ds.Access.Lock()
	ds.RequestStatus[action].Submitted = false
	ds.Access.Unlock()
}

func (ds *Interface) performAction(action uint8, param uint8, data []byte) (*SerialCanRequest, error) {
	ds.Access.Lock()
	defer ds.Access.Unlock()

	if !ds.Connected {
		return nil, InterfaceDisconnectedError
	}

	if ds.RequestStatus[action].Submitted {
		return nil, InterfaceBusyError
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

func (ds *Interface) SendSetPower(power uint8) (*SerialCanRequest, error) {
	return ds.performAction(SERIAL_CAN_CTRL_SET_POWER, power, nil)
}

func (ds *Interface) SendUpdatePowerStatus() (*SerialCanRequest, error) {
	return ds.performAction(SERIAL_CAN_CTRL_GET_POWER_STATUS, 0, nil)
}

/*
func (ds *Interface) SendReset() (*SerialCanRequest, error) {
	return ds.performAction(SERIAL_CAN_CTRL_RESET, 0, nil)
}
*/

func (ds *Interface) ProcessFrames(port *Port) {
	for {
		var frame CanFrame

		if ds.Serial.Recv(&frame) == false {
			clog.Error("Failed to receive frame from %s, will re-connect driver", ds.DeviceName)
			if ds.Rescue() {
				continue
			} else {
				break
			}
		}

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
					usbref := (uint16(frame.CanData[5]) << 8) | uint16(frame.CanData[6])
					powerlevel := (uint16(frame.CanData[1]) << 8) | uint16(frame.CanData[2])
					senselevel := (uint16(frame.CanData[3]) << 8) | uint16(frame.CanData[4])
					ds.PowerStatus.PowerOn = ((frame.CanData[0] & POWER_FLAGS_SUPPLY) != 0)
					ds.PowerStatus.SenseOn = ((frame.CanData[0] & POWER_FLAGS_SENSE) != 0)
					ds.PowerStatus.Fault = ((frame.CanData[0] & POWER_FLAGS_FAULT) != 0)
					ds.PowerStatus.PowerLevel = float32(powerlevel) / float32(usbref) * 1.1 * 7.2
					ds.PowerStatus.SenseLevel = 100 * float32(senselevel) / 1023
					ds.PowerStatus.UsbReference = 1023 * 1.1 / float32(usbref)
					errLevel := clog.INFO
					if ds.PowerStatus.Fault {
						errLevel = clog.WARNING
					}
					clog.Log(errLevel, "Power stat estimates: power=%t power_level=%.2fV sense=%t sense_level=%.3f%% fault=%t usb_power=%.2fV",
						ds.PowerStatus.PowerOn, ds.PowerStatus.PowerLevel,
						ds.PowerStatus.SenseOn, ds.PowerStatus.SenseLevel,
						ds.PowerStatus.Fault, ds.PowerStatus.UsbReference)
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

func (ds *Interface) ProcessMessages(port *Port) {
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

func (ds *Interface) Run(port *Port) {
	go ds.ProcessFrames(port)
	ds.SendUpdatePowerStatus()
	ds.ProcessMessages(port)
}

type InterfaceModel struct {
	Interfaces []*Interface
}

func NewInterfaceModel() *InterfaceModel {
	return &InterfaceModel{Interfaces: make([]*Interface, 0, 2)}
}

func (dm *InterfaceModel) Add(dr *Interface) {
	dm.Interfaces = append(dm.Interfaces, dr)
}

func (dm *InterfaceModel) GetInterface(id InterfaceId) *Interface {
	if int(id) >= len(dm.Interfaces) {
		return nil
	}
	return dm.Interfaces[id]
}

func (dm *InterfaceModel) Run(port *Port) {
	for _, driver := range dm.Interfaces {
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
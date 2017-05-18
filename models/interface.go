package models

import (
	"encoding/hex"
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
const (
	INTERFACE_RESISTOR_OFF = 1
	INTERFACE_RESISTOR_ON  = 0
)

var (
	InterfaceBusyError         = errors.New("Interface is busy, try again later")
	InterfaceDisconnectedError = errors.New("Interface is disconnected")
	InterfaceInputError        = errors.New("Interface input error")
	InterfaceTimeoutError      = errors.New("Operation timed out")
	InterfaceOperationError    = errors.New("Operation failed")
)

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

type InterfaceState struct {
	InterfaceId   int        `json:"id"`
	Access        sync.Mutex `json:"-"`
	Serial        *SerialCan `json:"-"`
	DeviceName    string     `json:"device_name"`
	InputResponse chan []byte
	InputBuffer   [128]*Message `json:"-"`
	Port          *Port
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

func newInterface(deviceName string) (*InterfaceState, error) {
	serial, err := SerialCanOpen(deviceName)
	if err != nil {
		clog.Error("Could not open %s: %s", deviceName, err.Error())
		return nil, err
	}
	//clog.Debug("Opened device %s", deviceName)

	driver := &InterfaceState{
		Serial:        serial,
		DeviceName:    deviceName,
		Connected:     true,
		InputResponse: make(chan []byte, 1)}

	return driver, nil
}

func (ds *InterfaceState) doCommand(query []byte) ([]byte, error) {
	if _, err := ds.Serial.Write(query[:]); err != nil {
		return nil, err
	}

	timeout := time.NewTimer(3 * time.Second)

	select {
	case result := <-ds.InputResponse:
		timeout.Stop()
		if (result[0] & 0xF0) == SERIAL_HEADER_SUCCESS {
			return result, nil
		}
		if result[0] == SERIAL_HEADER_FAIL {
			return nil, fmt.Errorf("Failed command: %s", hex.EncodeToString(query))
		}
		return nil, fmt.Errorf("Unexpected response from interface: %s", hex.EncodeToString(result))
	case <-timeout.C:
		return nil, fmt.Errorf("Timeout waiting for response from %s", ds.DeviceName)
	}
	return nil, nil // never reached
}

func (ds *InterfaceState) DoSoftReset() error {
	_, err := ds.doCommand([]byte{SERIAL_HEADER_REQUEST_SOFT_RESET})
	return err
}
func (ds *InterfaceState) DoHardReset() error {
	_, err := ds.doCommand([]byte{SERIAL_HEADER_REQUEST_HARD_RESET})
	return err
}
func (ds *InterfaceState) DoRequestPowerStatus() error {
	response, err := ds.doCommand([]byte{SERIAL_HEADER_REQUEST_POWER_STATUS})
	if err != nil {
		return err
	}
	usbref := (uint16(response[6]) << 8) | uint16(response[7])
	powerlevel := (uint16(response[2]) << 8) | uint16(response[3])
	senselevel := (uint16(response[4]) << 8) | uint16(response[5])
	ds.PowerStatus.PowerOn = ((response[1] & POWER_FLAGS_SUPPLY) != 0)
	ds.PowerStatus.SenseOn = ((response[1] & POWER_FLAGS_SENSE) != 0)
	ds.PowerStatus.Fault = ((response[1] & POWER_FLAGS_FAULT) != 0)
	ds.PowerStatus.SenseLevel = 100 * float32(senselevel) / 1023
	if usbref > 0 {
		ds.PowerStatus.PowerLevel = float32(powerlevel) / float32(usbref) * 1.1 * 7.2
		ds.PowerStatus.UsbReference = 1023 * 1.1 / float32(usbref)
	} else {
		ds.PowerStatus.PowerLevel = 0
		ds.PowerStatus.UsbReference = 0
	}
	errLevel := clog.INFO
	if ds.PowerStatus.Fault {
		errLevel = clog.WARNING
	}
	clog.Log(errLevel, "Power stat estimates: power=%t power_level=%.2fV sense=%t sense_level=%.3f%% fault=%t usb_power=%.2fV",
		ds.PowerStatus.PowerOn, ds.PowerStatus.PowerLevel,
		ds.PowerStatus.SenseOn, ds.PowerStatus.SenseLevel,
		ds.PowerStatus.Fault, ds.PowerStatus.UsbReference)
	return nil
}

func (ds *InterfaceState) DoSetPower(powerOn byte) error {
	_, err := ds.doCommand([]byte{SERIAL_HEADER_SET_POWER, powerOn})
	return err
}

func (ds *InterfaceState) DoSetCanResistor(resOn byte) error {
	// inverted logic
	_, err := ds.doCommand([]byte{SERIAL_HEADER_SET_CAN_RES, resOn})
	return err
}

func (ds *InterfaceState) DoVersion() ([]byte, error) {
	return ds.doCommand([]byte{SERIAL_HEADER_VERSION})
}

func (ds *InterfaceState) DoFrame(frame *CanFrame) error {
	packet, _ := frame.MarshalBinary()
	_, err := ds.doCommand(packet)
	return err
}

func (ds *InterfaceState) Rescue() bool {
	ds.Close()
	for {
		close(ds.InputResponse)
		ds.InputResponse = make(chan []byte, 1)
		serial, err := SerialCanOpen(ds.DeviceName)
		if err == nil {
			ds.Serial = serial
			clog.Info("Reopened device %s", ds.DeviceName)
			if err = ds.DoSoftReset(); err != nil {
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

func (ds *InterfaceState) Close() {
	ds.Serial.Close()
}

func (ds *InterfaceState) assemblePacket(packet []byte) error {
	var frame CanFrame

	if err := frame.UnmarshalBinary(packet); err != nil {
		clog.Error("Failed to unmarshall CAN frame from packet %s", hex.EncodeToString(packet))
		return err
	}

	node := frame.CanId.GetNode()

	switch {
	case !frame.CanId.IsExtended(), frame.CanId.IsRemote():
		clog.Warning("Got malformed frame, discarding.")
		return nil
	case frame.CanId.IsError():
		clog.Error("Recieved error frame on CAN controller")
		return nil
	default:
		if frame.CanId.IsFirst() {
			if ds.InputBuffer[node] != nil {
				clog.Warning("Got frame with inconsistent first bit indicator, discarding.")
				return nil
			}
			ds.InputBuffer[node] = NewMessageFromFrame(&frame)
		} else {
			if ds.InputBuffer[node] == nil {
				clog.Warning("Got first frame with missing first bit indicator, discarding.")
				return nil
			}
			ds.InputBuffer[node].AppendData(frame.CanData[:frame.CanDlc])
		}
		if frame.CanId.IsLast() {

			ds.Port.SendMessage(ds.InputBuffer[node])
			ds.InputBuffer[node] = nil
		}
	}
	clog.Debug("Got CAN frame: %s", frame.String())
	return nil
}

func (ds *InterfaceState) processInput() {
	for {
		packet := make([]byte, 16)
		_, err := ds.Serial.Read(packet)
		if err != nil {
			clog.Error("Failed to read from serial interface: %s", err.Error())
			ds.Rescue()
		} else {
			if packet[0] == SERIAL_HEADER_PACKET {
				ds.assemblePacket(packet)
			} else {
				select {
				case ds.InputResponse <- packet:
					// OK
				default:
					clog.Error("Unprocessed data from interface, rescuing.")
					ds.Rescue()
				}
			}
		}
	}
}

func (ds *InterfaceState) processMessages() {
	ticker := time.NewTicker(10 * time.Second)

	for {
		var frame CanFrame

		select {
		case m := <-ds.Port.Input:
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
				clog.Debug("Sending CAN frame: %s:", frame.String())
				if err := ds.DoFrame(&frame); err != nil {
					clog.Error("Failed to send frame to %s: %s", ds.DeviceName, err.Error())
					return
				}
				pos += int(frame.CanDlc)
				if pos >= len(m.Data) {
					break
				}
			}
		case <-ticker.C:
			/*
				if status, ok := ds.Serial.Status(); ok {
					clog.Debug("Serial status is %x", status)
				} else {
					clog.Error("Serial status failed")
				}
			*/
			ds.DoRequestPowerStatus()
		}
	}
}

type InterfaceModel struct {
	Interfaces []*InterfaceState
}

func NewInterfaceModel() *InterfaceModel {
	return &InterfaceModel{Interfaces: make([]*InterfaceState, 0, 2)}
}

func (dm *InterfaceModel) AddInterface(name string) (int, error) {
	dr, err := newInterface(name)
	if err != nil {
		return -1, err
	}
	dr.InterfaceId = len(dm.Interfaces)
	dr.Port = PortManager.CreatePort(fmt.Sprintf("interface-%d", dr.InterfaceId))
	dm.Interfaces = append(dm.Interfaces, dr)
	return dr.InterfaceId, nil
}

func (dm *InterfaceModel) GetInterface(id int) *InterfaceState {
	if int(id) >= len(dm.Interfaces) {
		return nil
	}
	return dm.Interfaces[id]
}

func (dm *InterfaceModel) Run() {
	for _, driver := range dm.Interfaces {
		go driver.processInput()
		go driver.processMessages()

		if err := driver.DoSoftReset(); err != nil {
			clog.Error(err.Error())
			panic(err.Error())
		}
	}
}

func (dm *InterfaceModel) Each(fn func(int, *InterfaceState)) {
	for _, driver := range dm.Interfaces {
		fn(driver.InterfaceId, driver)
	}
}

/*

for the controller:

GET /api/driver/:id
POST /api/driver/:id/control?c=reset
POST /api/driver/:id/control?c=poweron
POST /api/driver/:id/control?c=poweroff
*/

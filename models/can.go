package models

import (
	"fmt"
)

type CanId uint32

type CanFrame struct {
	CanId
	CanDlc  uint8
	CanData [8]uint8
}

func (frame *CanFrame) String() string {
	dlc := frame.CanDlc

	s := fmt.Sprintf("[%s ", frame.CanId.String())
	if frame.CanDlc > 8 {
		s += "!(dlc>8)"
		dlc = 8
	}
	s += fmt.Sprintf("%d:", frame.CanDlc)

	for i := uint8(0); i < dlc; i++ {
		s += fmt.Sprintf(" %02x", frame.CanData[i])
	}
	if frame.IsSystem() {
		s += " - " + NocanSysFuncString(frame.GetSysFunc())
	}
	return s + "]"
}

func (frame *CanFrame) UnmarshalBinary(p []byte) error {
	if len(p) != 16 {
		return fmt.Errorf("CanFrame.Load: Data block must be 16 bytes, got %d", len(p))
	}
	if p[0] != SERIAL_HEADER_PACKET {
		return fmt.Errorf("CanFrame.Load: Bad header, expected %x, got %x", SERIAL_HEADER_PACKET, p[0])
	}
	frame.CanId = (CanId(p[3]) << 24) | (CanId(p[4]) << 16) | (CanId(p[5]) << 8) | CanId(p[6])
	frame.CanDlc = uint8(p[7])
	if frame.CanDlc > 8 {
		return fmt.Errorf("CanFrame.Load: DLC must be less or equal to 8, got %d", frame.CanDlc)
	}
	for i := 0; i < 8; i++ {
		frame.CanData[i] = uint8(p[8+i])
	}
	return nil
}

func (frame *CanFrame) MarshalBinary() (data []byte, err error) {
	err = nil
	data = make([]byte, 16)
	data[0] = SERIAL_HEADER_PACKET
	data[1] = 0
	data[2] = 0
	data[3] = uint8(frame.CanId >> 24)
	data[4] = uint8(frame.CanId >> 16)
	data[5] = uint8(frame.CanId >> 8)
	data[6] = uint8(frame.CanId)
	data[7] = uint8(frame.CanDlc)
	for i := 0; i < 8; i++ {
		data[8+i] = uint8(frame.CanData[i])
	}
	return
}

const (
	CANID_MASK_EXTENDED = (1 << 31)
	CANID_MASK_REMOTE   = (1 << 30)
	CANID_MASK_ERROR    = (1 << 29)
	CANID_MASK_CONTROL  = (CANID_MASK_EXTENDED | CANID_MASK_REMOTE | CANID_MASK_ERROR)
	CANID_MASK_FIRST    = (1 << 28)
	CANID_MASK_LAST     = (1 << 20)
	CANID_MASK_SYSTEM   = (1 << 18)
	CANID_MASK_MESSAGE  = ^(CanId((1 << 28) | (1 << 20)))
)

func (canid CanId) IsFirst() bool {
	return (canid & CANID_MASK_FIRST) != 0
}

func (canid CanId) IsLast() bool {
	return (canid & CANID_MASK_LAST) != 0
}

func (canid CanId) IsExtended() bool {
	return (canid & CANID_MASK_EXTENDED) != 0
}

func (canid CanId) IsError() bool {
	return (canid >> CANID_MASK_ERROR) != 0
}

func (canid CanId) IsRemote() bool {
	return (canid >> CANID_MASK_REMOTE) != 0
}

func (canid CanId) IsControl() bool {
	return (canid & CANID_MASK_CONTROL) == CANID_MASK_CONTROL
}

func (canid CanId) IsSystem() bool {
	return (canid & CANID_MASK_SYSTEM) != 0
}

func (canid CanId) IsPublish() bool {
	return !canid.IsSystem() && !canid.IsControl()
}

func (canid CanId) GetNode() Node {
	return Node((canid >> 21) & 0x7F)
}

func (canid CanId) SetNode(node Node) CanId {
	canid &= ^(CanId(0x7F << 21))
	canid |= CanId(node) << 21
	return canid
}

func (canid CanId) GetSysFunc() uint8 {
	return uint8(canid >> 8)
}

func (canid CanId) SetSysFunc(fn uint8) CanId {
	canid &= CanId(0xFFFF00FF)
	canid |= CanId(fn) << 8
	return canid
}

func (canid CanId) GetSysParam() uint8 {
	return uint8(canid)
}

func (canid CanId) SetSysParam(pm uint8) CanId {
	canid &= CanId(0xFFFFFF00)
	canid |= CanId(pm)
	return canid
}

func (canid CanId) GetChannel() Channel {
	return Channel(canid & 0xFFFF)
}

func (canid CanId) SetChannel(channel Channel) CanId {
	canid &= CanId(0xFFFC0000)
	canid |= CanId(channel)
	return canid
}

func (canid CanId) String() string {
	s := fmt.Sprintf("<%08x,n=%d", uint(canid), canid.GetNode())
	if canid.IsFirst() {
		s += ",first"
	}
	if canid.IsLast() {
		s += ",last"
	}
	if canid.IsControl() {
		s += fmt.Sprintf(",ctrl,fn=%d,pm=%d", canid.GetSysFunc(), canid.GetSysParam())
	} else if canid.IsSystem() {
		s += fmt.Sprintf(",sys,fn=%d,pm=%d", canid.GetSysFunc(), canid.GetSysParam())
	} else {
		s += fmt.Sprintf(",pub,channel=%d", canid.GetChannel())
	}
	return s + ">"
}

var sys_func_strings = [...]string{
	"NOCAN_SYS_ANY",
	"NOCAN_SYS_ADDRESS_REQUEST",
	"NOCAN_SYS_ADDRESS_CONFIGURE",
	"NOCAN_SYS_ADDRESS_CONFIGURE_ACK",
	"NOCAN_SYS_ADDRESS_LOOKUP",
	"NOCAN_SYS_ADDRESS_LOOKUP_ACK",
	"NOCAN_SYS_NODE_BOOT_REQUEST",
	"NOCAN_SYS_NODE_BOOT_ACK",
	"NOCAN_SYS_NODE_PING",
	"NOCAN_SYS_NODE_PING_ACK",
	"NOCAN_SYS_CHANNEL_REGISTER",
	"NOCAN_SYS_CHANNEL_REGISTER_ACK",
	"NOCAN_SYS_CHANNEL_UNREGISTER",
	"NOCAN_SYS_CHANNEL_UNREGISTER_ACK",
	"NOCAN_SYS_CHANNEL_SUBSCRIBE",
	"NOCAN_SYS_CHANNEL_UNSUBSCRIBE",
	"NOCAN_SYS_CHANNEL_LOOKUP",
	"NOCAN_SYS_CHANNEL_LOOKUP_ACK",
	"NOCAN_SYS_BOOTLOADER_GET_SIGNATURE",
	"NOCAN_SYS_BOOTLOADER_GET_SIGNATURE_ACK",
	"NOCAN_SYS_BOOTLOADER_SET_ADDRESS",
	"NOCAN_SYS_BOOTLOADER_SET_ADDRESS_ACK",
	"NOCAN_SYS_BOOTLOADER_WRITE",
	"NOCAN_SYS_BOOTLOADER_WRITE_ACK",
	"NOCAN_SYS_BOOTLOADER_READ",
	"NOCAN_SYS_BOOTLOADER_READ_ACK",
	"NOCAN_SYS_BOOTLOADER_LEAVE",
	"NOCAN_SYS_BOOTLOADER_LEAVE_ACK",
}

func NocanSysFuncString(fn uint8) string {
	if fn >= uint8(len(sys_func_strings)) {
		return "!ERR!"
	}
	return sys_func_strings[fn]
}

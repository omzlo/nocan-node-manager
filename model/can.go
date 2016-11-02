package model

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
	s := fmt.Sprintf("[%s %d:", frame.CanId.String(), frame.CanDlc)
	for i := uint8(0); i < frame.CanDlc; i++ {
		s += fmt.Sprintf(" %02x", frame.CanData[i])
	}
	if frame.IsSystem() {
		s += " - " + NocanSysFuncString(frame.GetSysFunc())
	}
	return s + "]"
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

func (canid CanId) GetTopic() Topic {
	var i uint8
	var base uint8 = uint8(((canid >> 16) & 0x03) << 4)
	for i = 0; i < 16; i++ {
		if (canid & (1 << i)) != 0 {
			return Topic(base + i)
		}
	}
	return -1
}

func (canid CanId) SetTopic(topic Topic) CanId {
	t := ((uint32(topic) >> 4) << 16) | (1 << (uint32(topic) & 0xF))
	canid &= CanId(0xFFFC0000)
	canid |= CanId(t)
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
		s += fmt.Sprintf(",pub,topic=%d", canid.GetTopic())
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
	"NOCAN_SYS_TOPIC_REGISTER",
	"NOCAN_SYS_TOPIC_REGISTER_ACK",
	"NOCAN_SYS_TOPIC_UNREGISTER",
	"NOCAN_SYS_TOPIC_UNREGISTER_ACK",
	"NOCAN_SYS_TOPIC_SUBSCRIBE",
	"NOCAN_SYS_TOPIC_UNSUBSCRIBE",
	"NOCAN_SYS_TOPIC_LOOKUP",
	"NOCAN_SYS_TOPIC_LOOKUP_ACK",
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

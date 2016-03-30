package game

import (
	"../messages/room_messages/proto_files"
)

const maxCommandCount int = 16

// +gen * slice:"First"
type Player struct {
	Id string
	Index int32
	SessionChan *UdpSessioin
	nextMsgId int32
	lastFrame int32
	commands []*messages.GenMessage
}

func NewPlayer()  *Player{
	return &Player{commands:make([]*messages.GenMessage, 0, maxCommandCount)}
}
func (this *Player) GetComand(msg *messages.GenMessage)  bool{
	i := 0
	for i = len(this.commands) - 1; i >= 0; i--{
		item := this.commands[i]
		diff := *item.Frame - *msg.Frame
		if diff < 0 {
			newSlice := append(this.commands, nil)
			copy(newSlice[i + 1:], newSlice[i:])
			newSlice[i] = msg
			this.commands = newSlice
			return true
		}else if diff == 0{
			return *item.MsgId != *msg.MsgId
		}
	}
	if i == -1{
		newSlice := this.commands[0 : len(this.commands)+1]
		copy(newSlice[1:], newSlice[0:])
		newSlice[0] = msg
		this.commands = newSlice
		return true
	}
	return false
}

func (this *Player) TryRemoveHeadCmd()  {
	if len(this.commands) > maxCommandCount{
		this.commands = this.commands[1:]
	}
}
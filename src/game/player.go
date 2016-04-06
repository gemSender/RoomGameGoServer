package game


const maxCommandCount int = 16

type playerCommand struct{
	frame int32
	msgId int32
	bytes []byte
}

// +gen * slice:"First"
type Player struct {
	Id string
	Index int32
	SessionChan PlayerRoomSession
	nextMsgId int32
	lastFrame int32
	commands []*playerCommand
}

func NewPlayer()  *Player{
	return &Player{commands:make([]*playerCommand, 0, maxCommandCount)}
}
func (this *Player) GetComand(msgId int32, frame int32, msgBytes []byte)  bool{
	i := 0
	for i = len(this.commands) - 1; i >= 0; i--{
		item := this.commands[i]
		diff := item.frame - frame
		if diff < 0 {
			newSlice := append(this.commands, nil)
			copy(newSlice[i + 1:], newSlice[i:])
			newSlice[i] = &playerCommand{frame:frame, bytes:msgBytes, msgId : msgId}
			this.commands = newSlice
			return true
		}else if diff == 0{
			return item.msgId != msgId
		}
	}
	if i == -1{
		newSlice := this.commands[0 : len(this.commands)+1]
		copy(newSlice[1:], newSlice[0:])
		newSlice[0] = &playerCommand{frame:frame, bytes:msgBytes, msgId : msgId}
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
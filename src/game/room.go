package game

import (
	"github.com/golang/protobuf/proto"
	"../messages/proto_files"
	"../messages/room_messages/proto_files"
	"../messages/world_messages/proto_files"
	"log"
)

// +gen * slice:"First"
type Room struct {
	Id int32
	PlayerCount int32
	Capacity int32
	Players PlayerSlice
	maxFrame int32
}

func NewRoom(capacity int32) *Room {
	ret := &Room{}
	ret.Players = make(PlayerSlice, 0, 4)
	ret.Capacity = capacity
	return ret
}

func (this *Room) ProcessCommand(session PlayerRoomSession, playerIndex int32, msgId int32, frame int32, msgBytes []byte) {
	if this.maxFrame < frame {
		this.maxFrame = frame
	}
	player, _ := this.Players.First(func(item *Player) bool{
		return item.Index == playerIndex
	})
	if player.GetComand(msgId, frame, msgBytes){
		player.TryRemoveHeadCmd()
		player.SessionChan = session
		if player.nextMsgId <= msgId{
			player.nextMsgId = msgId
		}
		for i, imax := 0, len(this.Players); i < imax; i++{
			sess := this.Players[i].SessionChan
			if sess != nil{
				sess.Send(msgBytes)
			}
		}
	}else{
		log.Println("Ignore Command of ", player.Id, " at frame", frame)
	}
}

func (this *Room) AddPlayer(playerId string, playerIndex int32) (world_messages.EnterRoomResult, []int32) {
	_, findErr := this.Players.First(func(x *Player) bool{
		return x.Index == playerIndex
	})
	if findErr != nil{
		if len(this.Players) > int(this.Capacity){
			return world_messages.EnterRoomResult_OutOfCapacity, nil
		}
		newPlayer := NewPlayer()
		newPlayer.Id = playerId
		newPlayer.Index = playerIndex
		newPlayer.nextMsgId = 1
		this.Players = append(this.Players, newPlayer)
		allPlayerIndices := make([]int32, len(this.Players))
		for i, p := range this.Players {
			allPlayerIndices[i] = p.Index
		}
		return world_messages.EnterRoomResult_Ok, allPlayerIndices
	}else {
		return world_messages.EnterRoomResult_AlreadyIn, nil
	}
}

func (this *Room) GetCommand(playerIndex int32, frame int32) *playerCommand{
	playerItem, findErr := this.Players.First(func(x *Player) bool{
		return x.Index == playerIndex
	})
	if findErr == nil {
		for _, c := range playerItem.commands {
			if c.frame == frame{
				return c
			}
		}
	}
	return nil
}

func (this *Room) RemovePlayer(playerId string) {
	index := -1
	for i, p := range this.Players{
		if p.Id == playerId {
			index = i
			break
		}
	}
	if index >= 0{
		count := len(this.Players)
		if index == count {
			copy(this.Players[index:], this.Players[index + 1:])
		}
		this.Players = this.Players[:count - 1]
	}
}

func (room *Room)Rpc(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte, innerMsg proto.Message)  {
	rpcMsg := innerMsg.(*room_messages.Rpc)
	room.ProcessCommand(session, msg.GetPIdx(), msg.GetMsgId(), rpcMsg.GetFrame(), msgBytes)
}

func (room *Room) CreateObj(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte, innerMsg proto.Message)  {
	crtObjMsg := innerMsg.(*room_messages.CreateObj)
	room.ProcessCommand(session, msg.GetPIdx(), msg.GetMsgId(), crtObjMsg.GetFrame(), msgBytes)
}

func (room *Room) ReadyForGame(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte, innerMsg proto.Message) {
	rdyForGame := innerMsg.(*room_messages.ReadyForGame)
	room.ProcessCommand(session, msg.GetPIdx(), msg.GetMsgId(), rdyForGame.GetFrame(), msgBytes)
}

func (room *Room) Empty(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte, innerMsg proto.Message)  {
	emptyMsg := innerMsg.(*room_messages.Empty)
	room.ProcessCommand(session, msg.GetPIdx(), msg.GetMsgId(), emptyMsg.GetFrame(), msgBytes)
}

func (room *Room) GetMissingCmd(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte, innerMsg proto.Message)  {
	getMissingCmdMsg := innerMsg.(*room_messages.GetMissingCmd)
	targetCmd := room.GetCommand(getMissingCmdMsg.GetPlayerIndex(), getMissingCmdMsg.GetFrame())
	if targetCmd != nil{
		session.Send(targetCmd.bytes)
	}
}
package game

import (
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

func (this *Room) ProcessCommand(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte) {
	frame := *msg.Frame
	if this.maxFrame < frame {
		this.maxFrame = frame
	}
	playerIndex := *msg.PIdx
	player, _ := this.Players.First(func(item *Player) bool{
		return item.Index == playerIndex
	})
	if player.GetComand(msg){
		player.TryRemoveHeadCmd()
		player.SessionChan = session
		msgId := *msg.MsgId
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
		log.Println("Ignore Command of ", player.Id, " at frame", *msg.Frame)
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

func (this *Room) GetCommand(playerIndex int32, frame int32) *messages.GenMessage{
	playerItem, findErr := this.Players.First(func(x *Player) bool{
		return x.Index == playerIndex
	})
	if findErr == nil {
		for _, c := range playerItem.commands {
			if c.GetFrame() == frame{
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
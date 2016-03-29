package game

import (
	"../messages/world_messages/proto_files"
	"../messages/room_messages/proto_files"
)

var world_instance *World

type World struct {
	Rooms RoomSlice
	PlayerId_Index_Map map[string]int32
	PlayerIndex_Room_Map map[int32]*Room
}

func CreateWorld()  *World{
	ret := &World{}
	ret.Rooms = make(RoomSlice, 0, 32)
	ret.PlayerId_Index_Map = make(map[string]int32)
	ret.PlayerIndex_Room_Map = make(map[int32]*Room)
	world_instance = ret
	return ret
}

func (this *World) CreateRoom(capacity int32) *Room{
	ret := NewRoom(capacity)
	this.Rooms = append(this.Rooms, ret)
	return ret
}

func (this *World) EnterRoom(playerId string, playerIndex int32, roomIdx int32) (world_messages.EnterRoomResult, []int32) {
	room, findErr := this.Rooms.First(func(x *Room) bool{
		return x.Id == roomIdx
	})
	if findErr == nil{
		this.PlayerIndex_Room_Map[playerIndex] = room
		return room.AddPlayer(playerId, playerIndex)
	}
	return world_messages.EnterRoomResult_RoomNotExists, nil
}

func (this *World) ProcessCommand(playerIndex int32, sessionChan chan []byte, msg *messages.GenMessage)  {
	room, find :=this.PlayerIndex_Room_Map[playerIndex]
	if find {
		room.GetMessage(sessionChan, msg)
	}
}

func (this *World) GetRoomByPlayerIndex(index int32)  *Room{
	return this.PlayerIndex_Room_Map[index]
}

func (this *World) GetRoomByRoomIndex(index int32) *Room{
	ret, _ := this.Rooms.First(func(x *Room) bool {
		return x.Id == index
	})
	return ret
}



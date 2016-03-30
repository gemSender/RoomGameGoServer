package game

import (
	"../messages/world_messages/proto_files"
	"../utility"
	"github.com/golang/protobuf/proto"
	"log"
)

var world_instance *World

type World struct {
	Rooms RoomSlice
	PlayerId_Index_Map map[string]int32
	PlayerIndex_Room_Map map[int32]*Room
	nextRoomId int32
	nextPlayerIndex int32
}

func CreateWorld()  *World{
	ret := &World{}
	ret.Rooms = make(RoomSlice, 0, 32)
	ret.PlayerId_Index_Map = make(map[string]int32)
	ret.PlayerIndex_Room_Map = make(map[int32]*Room)
	ret.nextRoomId = 1
	ret.nextPlayerIndex = 1
	world_instance = ret
	worldMsgDispatcher.RegisterHandler(world_messages.MessageType_CreateRoom, ret.OnCreateRoomMsgCallBack)
	worldMsgDispatcher.RegisterHandler(world_messages.MessageType_EnterRoom, ret.OnEnterRoomCallBack)
	worldMsgDispatcher.RegisterHandler(world_messages.MessageType_GetRoomList, ret.OnGetRoomList)
	return ret
}

func (this *World) CreateRoom(capacity int32) *Room{
	ret := NewRoom(capacity)
	roomId := this.nextRoomId
	ret.Id = roomId
	this.nextRoomId ++
	this.Rooms = append(this.Rooms, ret)
	return ret
}

func (this *World) EnterRoom(playerId string, pIndex *int32,roomIdx int32) (world_messages.EnterRoomResult, []int32) {
	room, findErr := this.Rooms.First(func(x *Room) bool{
		return x.Id == roomIdx
	})
	if findErr == nil{
		playerIndex := this.nextPlayerIndex
		this.nextPlayerIndex ++
		this.PlayerIndex_Room_Map[playerIndex] = room
		*pIndex = playerIndex
		return room.AddPlayer(playerId, playerIndex)
	}
	return world_messages.EnterRoomResult_RoomNotExists, nil
}

/*
func (this *World) ProcessCommand(playerIndex int32, sessionChan chan []byte, msg *messages.GenMessage)  {
	room, find :=this.PlayerIndex_Room_Map[playerIndex]
	if find {
		room.ProcessCommand(sessionChan, msg)
	}
}
*/

func (this *World) GetRoomByPlayerIndex(index int32)  *Room{
	return this.PlayerIndex_Room_Map[index]
}

func (this *World) GetRoomByRoomIndex(index int32) *Room{
	ret, _ := this.Rooms.First(func(x *Room) bool {
		return x.Id == index
	})
	return ret
}

func (this *World)  OnPlayerDisconnect(playerId string){
	Index, findIndex := this.PlayerId_Index_Map[playerId]
	if findIndex{
		_, findRoom := this.PlayerIndex_Room_Map[Index]
		if findRoom{
			// exit room
		}
	}
}

func (this *World) OnCreateRoomMsgCallBack(session WorldSession, msg *world_messages.WorldMessage, msgBytes []byte) {
	buff := msg.GetBuff()
	capacity := utility.BytesToInt32(buff)
	newRoom := this.CreateRoom(capacity)
	replyMsg := world_messages.MsgCreateRoomReply{
		Capacity:&capacity,
		Id:&newRoom.Id,
	}
	replyBytes, encodeErr := proto.Marshal(&replyMsg)
	if encodeErr != nil{
		log.Panic(encodeErr)
	}
	session.Send(world_messages.MessageType_CreateRoomReply, msg.GetMsgId(), replyBytes)
}

func (this *World) OnEnterRoomCallBack(session WorldSession, msg *world_messages.WorldMessage, msgBytes []byte) {
	buff := msg.GetBuff()
	roomId := utility.BytesToInt32(buff)
	var myIndex int32
	result, playerIdices := this.EnterRoom(session.PlayerId, &myIndex, roomId)
	replyMsg := world_messages.MsgEnterRoomReply{}
	replyMsg.Result = &result
	replyMsg.AllockedIndex = &myIndex
	if result == world_messages.EnterRoomResult_Ok{
		replyMsg.Players = playerIdices
		pushMsg := world_messages.MsgPlayerEnterRoom{}
		pushMsg.RoomId = &roomId
		pushMsg.PlayerIndex = &myIndex
		pushBytes, encodeErr := proto.Marshal(&pushMsg)
		if encodeErr != nil{
			panic(encodeErr)
		}
		for pId, pSession := range worldMsgDispatcher.playerChanDict{
			if pId != session.PlayerId {
				pSession.Send(world_messages.MessageType_PlayerEnterRoom, -1, pushBytes)
			}
		}
	}
	replyBytes, encodeErr1 := proto.Marshal(&replyMsg)
	if encodeErr1 != nil{
		panic(encodeErr1)
	}
	session.Send(world_messages.MessageType_EnterRoomReply, msg.GetMsgId(), replyBytes)
}

func (this *World) OnGetRoomList(session WorldSession, msg *world_messages.WorldMessage, msgBytes []byte) {
	replyRooms := make([]*world_messages.Room, len(this.Rooms))
	replyMsg := world_messages.MsgGetRoomListReply{}
	for i, r := range this.Rooms{
		item := &world_messages.Room{}
		rid, cap, pCount := r.Id, r.Capacity, int32(len(r.Players))
		item.Capacity = &cap
		item.Id = &rid
		item.PlayerCount = &pCount
		replyRooms[i] = item
	}
	replyMsg.Rooms = replyRooms
	replyBytes, encodeErr := proto.Marshal(&replyMsg)
	if encodeErr != nil{
		log.Panic(encodeErr)
	}
	session.Send(world_messages.MessageType_GetRoomListReply, msg.GetMsgId(), replyBytes)
}
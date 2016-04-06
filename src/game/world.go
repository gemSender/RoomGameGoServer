package game

import (
	world_messages "../messages/world_messages/proto_files"
	"log"
	"github.com/golang/protobuf/proto"
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
	return ret
}

func (this *World) DoCreateRoom(capacity int32) *Room{
	ret := NewRoom(capacity)
	roomId := this.nextRoomId
	ret.Id = roomId
	this.nextRoomId ++
	this.Rooms = append(this.Rooms, ret)
	return ret
}

func (this *World) DoEnterRoom(playerId string, pIndex *int32,roomIdx int32) (world_messages.EnterRoomResult, []int32) {
	room, findErr := this.Rooms.First(func(x *Room) bool{
		return x.Id == roomIdx
	})
	if findErr == nil{
		playerIndex := this.nextPlayerIndex
		this.nextPlayerIndex ++
		this.PlayerIndex_Room_Map[playerIndex] = room
		this.PlayerId_Index_Map[playerId] = playerIndex
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
		delete(this.PlayerId_Index_Map, playerId)
		room, findRoom := this.PlayerIndex_Room_Map[Index]
		if findRoom{
			log.Println("player quit room")
			delete(this.PlayerIndex_Room_Map, Index)
			room.RemovePlayer(playerId)
			pushMsg := &world_messages.PlayerQuitRoom{PlayerId:&playerId, RoomId:&room.Id, PlayerIndex:&Index}
			worldMsgDispatcher.PushMutiply(pushMsg, func(pId string) bool {return pId != playerId})
		}
	}
}

func (this *World) CreateRoom(session WorldSession, msgId int32, innerMsg proto.Message) {
	capacity := innerMsg.(*world_messages.CreateRoom).GetCapacity()
	newRoom := this.DoCreateRoom(capacity)
	replyMsg := world_messages.CreateRoomReply{
		Capacity:&capacity,
		Id:&newRoom.Id,
	}
	session.Send(msgId, &replyMsg)
	pushMsg := world_messages.PlayerCreateRoom{RoomId:&newRoom.Id, PlayerId:&session.PlayerId, Capacity:&newRoom.Capacity}
	worldMsgDispatcher.PushMutiply(&pushMsg, func(pId string) bool{return pId != session.PlayerId})
}

func (this *World) EnterRoom(session WorldSession, msgId int32, innerMsg proto.Message) {
	roomId := innerMsg.(*world_messages.EnterRoom).GetRoomId()
	var myIndex int32
	result, playerIdices := this.DoEnterRoom(session.PlayerId, &myIndex, roomId)
	replyMsg := &world_messages.EnterRoomReply{}
	replyMsg.Result = &result
	replyMsg.AllockedIndex = &myIndex
	if result == world_messages.EnterRoomResult_Ok{
		replyMsg.Players = playerIdices
		pushMsg := &world_messages.PlayerEnterRoom{}
		pushMsg.RoomId = &roomId
		pushMsg.PlayerIndex = &myIndex
		worldMsgDispatcher.PushMutiply(pushMsg, func(pId string) bool {return pId != session.PlayerId})
	}
	session.Send(msgId, replyMsg)
}


func (this *World) GetRoomList(session WorldSession, msgId int32, innerMsg proto.Message) {
	replyRooms := make([]*world_messages.Room, len(this.Rooms))
	replyMsg := &world_messages.GetRoomListReply{}
	for i, r := range this.Rooms{
		item := &world_messages.Room{}
		rid, cap, pCount := r.Id, r.Capacity, int32(len(r.Players))
		item.Capacity = &cap
		item.Id = &rid
		item.PlayerCount = &pCount
		replyRooms[i] = item
	}
	replyMsg.Rooms = replyRooms
	session.Send(msgId, replyMsg)
}

func (this *World) Login(session WorldSession, msgId int32, innerMsg proto.Message) {

	loginMsg := innerMsg.(*world_messages.Login)
	if worldMsgDispatcher.GetSession(loginMsg.GetPlayerId()) == nil{
		session.PlayerId = loginMsg.GetPlayerId()
		worldMsgDispatcher.RegisterPlayer(session)
		var errorCode int32 = 0
		replyMsg := &world_messages.LoginReply{ErrorCode:&errorCode}
		session.Send(msgId, replyMsg)
	}else{
		var errorCode int32 = 1
		replyMsg := &world_messages.LoginReply{ErrorCode:&errorCode}
		session.Send(msgId, replyMsg)
	}
}
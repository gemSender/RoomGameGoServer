package game

import (
	"../messages/world_messages/proto_files"
	"github.com/golang/protobuf/proto"
	"log"
)
var worldMsgDispatcher *WorldMessageDispatcher;

type ClientMsg struct{
	SendChannel chan <- []byte
	Bytes []byte
	IdChannel chan <- string
}

type WorldSession struct {
	SendChannel chan <-[]byte
	PlayerId string
}

func (this *WorldSession) Send(msgType world_messages.MessageType, msgId int32, bytes []byte) {
	msg := world_messages.ReplyMsg{}
	msg.MsgId = &msgId
	msg.Buff = bytes
	msg.Type = &msgType
	msgBytes, encodeErr := proto.Marshal(&msg)
	if encodeErr != nil{
		log.Panic(encodeErr)
	}
	this.SendChannel <- msgBytes
}

type WorldMessageDispatcher struct{
	playerChanDict map[string]WorldSession
	msgActionDict map[world_messages.MessageType]func(WorldSession, *world_messages.WorldMessage, []byte)
}

func CreateWorldMessageDispatcher() *WorldMessageDispatcher{
	ret := &WorldMessageDispatcher{}
	ret.msgActionDict = make(map[world_messages.MessageType]func(WorldSession, *world_messages.WorldMessage, []byte))
	ret.playerChanDict = make(map[string]WorldSession)
	worldMsgDispatcher = ret
	return ret
}

func (this *WorldMessageDispatcher) RegisterHandler(msgType world_messages.MessageType, action func(WorldSession, *world_messages.WorldMessage, []byte))  {
	this.msgActionDict[msgType] = action
}

func (this *WorldMessageDispatcher) StartRecvLoop(exitNotifyWorldChan chan string, worldMsgChan <- chan ClientMsg)  {
	world := world_instance
	for {
		select {
		case exitPlayerId := <- exitNotifyWorldChan:
			delete(this.playerChanDict, exitPlayerId)
			world.OnPlayerDisconnect(exitPlayerId)
		case clientMsg := <- worldMsgChan:
			msg := &world_messages.WorldMessage{}
			parseErr := proto.Unmarshal(clientMsg.Bytes, msg)
			if parseErr != nil{
				panic(parseErr)
			}
			playerId := msg.GetPlayerId()
			session, _ := this.playerChanDict[playerId]
			if clientMsg.IdChannel != nil{
				clientMsg.IdChannel <- playerId
				session = WorldSession{PlayerId:playerId, SendChannel:clientMsg.SendChannel}
				this.playerChanDict[playerId] = session
			}
			log.Println("get client msg", world_messages.MessageType_name[int32(msg.GetType())])
			this.msgActionDict[msg.GetType()](session, msg, clientMsg.Bytes)
		}
	}
}
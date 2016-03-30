package game

import (
	"../messages/room_messages/proto_files"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"../utility"
)

type RoomMessageDispatcher struct {
	indexRoomMap map[int32]*Room
	msgTypeActionMap map[messages.MessageType]func(*UdpSessioin, *messages.GenMessage, []byte, *Room)
}

type ConnRoomMsg struct {
	RemoteAddr *net.UDPAddr
	Bytes []byte
}

type UdpSessioin struct {
	UdpConn    *net.UDPConn
	RemoteAddr *net.UDPAddr
}

func (this *UdpSessioin) Send(bytes []byte)  {
	this.UdpConn.WriteTo(bytes, this.RemoteAddr)
}

func CreateRoomMsgDispatcher()  *RoomMessageDispatcher{
	ret := &RoomMessageDispatcher{}
	ret.indexRoomMap = make(map[int32]*Room)
	ret.msgTypeActionMap = make(map[messages.MessageType]func(*UdpSessioin, *messages.GenMessage, []byte, *Room))
	ret.RegisterMessageHandler(messages.MessageType_Rpc, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_CreateObj, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_ReadyForGame, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_Empty, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_GetMissingCmd, HandleGetMissingCmd)
	return ret;
}

func (this *RoomMessageDispatcher) RegisterMessageHandler(msgType messages.MessageType, action func(*UdpSessioin, *messages.GenMessage, []byte, *Room)) {
	this.msgTypeActionMap[msgType] = action
}

func (this *RoomMessageDispatcher) StartReceiveLoop(recvChannel <- chan ConnRoomMsg, conn *net.UDPConn)  {
	for {
		conn_msg := <- recvChannel
		roomMsg := messages.GenMessage{}
		decodeErr := proto.Unmarshal(conn_msg.Bytes, &roomMsg)
		if decodeErr != nil{
			log.Panic(decodeErr)
		}
		session := &UdpSessioin{UdpConn:conn, RemoteAddr:conn_msg.RemoteAddr}
		playerIndex := roomMsg.GetPIdx()
		room := world_instance.GetRoomByPlayerIndex(playerIndex)
		if room != nil {
			this.msgTypeActionMap[roomMsg.GetMsgType()](session, &roomMsg, conn_msg.Bytes, room)
		}
	}
}

func HandleGenMessage(session *UdpSessioin, msg *messages.GenMessage, msgBytes []byte, room *Room)  {
	room.ProcessCommand(session, msg, msgBytes)
}

func HandleGetMissingCmd(session *UdpSessioin, msg *messages.GenMessage, msgBytes []byte, room *Room)  {
	buf := msg.Buf
	playerIndex := utility.BytesToInt32(buf)
	targetCmd := room.GetCommand(playerIndex, msg.GetFrame())
	if targetCmd != nil{
		cmdBytes, encodeErr := proto.Marshal(targetCmd)
		if encodeErr != nil{
			log.Panic(encodeErr)
		}
		session.Send(cmdBytes)
	}
}
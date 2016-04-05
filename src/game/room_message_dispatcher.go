package game

import (
	"../messages/room_messages/proto_files"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"../utility"
	"github.com/gorilla/websocket"
	"net/http"
)

type PlayerRoomSession interface {
	Send([]byte)
}

type RoomMessageDispatcher struct {
	wsMsgChan chan WebSocketMsg
	indexRoomMap map[int32]*Room
	msgTypeActionMap map[messages.MessageType]func(PlayerRoomSession, *messages.GenMessage, []byte, *Room)
}

type ConnRoomUdpMsg struct {
	RemoteAddr *net.UDPAddr
	Bytes []byte
}

type RoomUdpMsg struct {
	sendChan chan []byte
	Bytes []byte
}

type UdpSessioin struct {
	UdpConn    *net.UDPConn
	RemoteAddr *net.UDPAddr
}

func (this *UdpSessioin) Send(bytes []byte)  {
	this.UdpConn.WriteTo(bytes, this.RemoteAddr)
}

type WebsocketSession struct {
	SendChan chan []byte
}

func (this *WebsocketSession) Send(bytes []byte) {
	this.SendChan <- bytes
}

type WebSocketMsg struct {
	SendChan chan []byte
	Bytes []byte
}

func CreateRoomMsgDispatcher()  *RoomMessageDispatcher{
	ret := &RoomMessageDispatcher{}
	ret.indexRoomMap = make(map[int32]*Room)
	ret.msgTypeActionMap = make(map[messages.MessageType]func(PlayerRoomSession, *messages.GenMessage, []byte, *Room))
	ret.RegisterMessageHandler(messages.MessageType_Rpc, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_CreateObj, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_ReadyForGame, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_Empty, HandleGenMessage)
	ret.RegisterMessageHandler(messages.MessageType_GetMissingCmd, HandleGetMissingCmd)
	return ret;
}

func (this *RoomMessageDispatcher) RegisterMessageHandler(msgType messages.MessageType, action func(PlayerRoomSession, *messages.GenMessage, []byte, *Room)) {
	this.msgTypeActionMap[msgType] = action
}

func (this *RoomMessageDispatcher) StartReceiveLoop()  {
	udpListenAddr, _ := net.ResolveUDPAddr("udp", ":2012")
	udpConn, udpListenErr := net.ListenUDP("udp", udpListenAddr)
	if udpListenErr != nil{
		log.Println(udpListenErr)
	}
	roomChannel := make(chan ConnRoomUdpMsg)
	go PlayerRoomUdpMsgRecvLoop(roomChannel, udpConn)
	this.wsMsgChan = make(chan WebSocketMsg, 256)
	http.HandleFunc("/room", this.ServeRoomWS)
	for {
		select {
		case conn_msg := <- roomChannel:
			session := &UdpSessioin{UdpConn:udpConn, RemoteAddr:conn_msg.RemoteAddr}
			this.OnReceiveMessage(session, conn_msg.Bytes)
		case chan_msg := <- this.wsMsgChan:
			session := &WebsocketSession{SendChan:chan_msg.SendChan}
			this.OnReceiveMessage(session, chan_msg.Bytes)
		}
	}
}

func (this *RoomMessageDispatcher) OnReceiveMessage(session PlayerRoomSession, bytes []byte)  {
	roomMsg := messages.GenMessage{}
	decodeErr := proto.Unmarshal(bytes, &roomMsg)
	if decodeErr != nil{
		log.Panic(decodeErr)
	}
	playerIndex := roomMsg.GetPIdx()
	room := world_instance.GetRoomByPlayerIndex(playerIndex)
	if room != nil {
		this.msgTypeActionMap[roomMsg.GetMsgType()](session, &roomMsg, bytes, room)
	}
}


func HandleGenMessage(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte, room *Room)  {
	room.ProcessCommand(session, msg, msgBytes)
}

func HandleGetMissingCmd(session PlayerRoomSession, msg *messages.GenMessage, msgBytes []byte, room *Room)  {
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

func (this *RoomMessageDispatcher) ServeRoomWS(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin : func(r *http.Request) bool {return true},
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	exitChan := make(chan int)
	sendChan := make(chan []byte, 256)
	go WsRoomConnReceiveProcess(ws, exitChan, sendChan, this.wsMsgChan)
	go WsRoomConnWriteProcess(ws, exitChan, sendChan)
}

func PlayerRoomUdpMsgRecvLoop(roomChannel chan ConnRoomUdpMsg, conn *net.UDPConn)  {
	recvBuf := make([]byte, 256)
	for  {
		bcount, remoteAddr, error := conn.ReadFromUDP(recvBuf)
		if error != nil{
			panic(error)
		}
		copiedBytes := make([]byte, bcount)
		copy(copiedBytes, recvBuf[:bcount])
		msg := ConnRoomUdpMsg{RemoteAddr:remoteAddr, Bytes:copiedBytes}
		roomChannel <- msg
	}
}

func WsRoomConnReceiveProcess(conn *websocket.Conn, exitChan chan <- int, sendChan chan []byte, msgChan chan <- WebSocketMsg)  {
	defer func() {exitChan <- 1; conn.Close()}()
	for {
		msgType, bytes, readErr := conn.ReadMessage()
		if readErr != nil{
			log.Panic(readErr)
		}
		if msgType == websocket.CloseMessage{
			return
		}
		msgChan <- WebSocketMsg{SendChan:sendChan, Bytes:bytes}
	}
}

func WsRoomConnWriteProcess(conn *websocket.Conn, exitChan <- chan int, sendChan chan []byte)  {
	defer conn.Close()
	for {
		select {
		case bytes := <-sendChan:
			conn.WriteMessage(websocket.BinaryMessage, bytes)
		case <-exitChan:
			return
		}
	}
}
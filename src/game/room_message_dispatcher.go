package game

import (
	"../messages/proto_files"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"github.com/gorilla/websocket"
	"net/http"
	"reflect"
)

type PlayerRoomSession interface {
	Send([]byte)
}

type RoomMessageDispatcher struct {
	wsMsgChan chan WebSocketMsg
	roomChannel chan ConnRoomUdpMsg
	udpConn *net.UDPConn
	indexRoomMap map[int32]*Room
	msgTypeActionMap map[string]func(PlayerRoomSession, *messages.GenMessage, []byte, *Room)
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
	ret.msgTypeActionMap = make(map[string]func(PlayerRoomSession, *messages.GenMessage, []byte, *Room))
	return ret;
}


func (this *RoomMessageDispatcher) StartReceiveLoop()  {
	for {
		select {
		case conn_msg := <- this.roomChannel:
			session := &UdpSessioin{UdpConn:this.udpConn, RemoteAddr:conn_msg.RemoteAddr}
			this.OnReceiveMessage(session, conn_msg.Bytes)
		case chan_msg := <- this.wsMsgChan:
			session := &WebsocketSession{SendChan:chan_msg.SendChan}
			this.OnReceiveMessage(session, chan_msg.Bytes)
		}
	}
}

func (this *RoomMessageDispatcher) OnReceiveMessage(session PlayerRoomSession, bytes []byte)  {
	roomMsg := &messages.GenMessage{}
	decodeErr := proto.Unmarshal(bytes, roomMsg)
	if decodeErr != nil{
		log.Panic(decodeErr)
	}
	playerIndex := roomMsg.GetPIdx()
	room := world_instance.GetRoomByPlayerIndex(playerIndex)
	if room != nil {
		innerMsgValue := reflect.New(proto.MessageType(roomMsg.GetType()).Elem())
		decodeErr1 := proto.Unmarshal(roomMsg.GetBuf(), innerMsgValue.Interface().(proto.Message))
		if decodeErr1 != nil{
			log.Panic(decodeErr1)
		}
		reflect.ValueOf(room).MethodByName(roomMsg.GetType()[len("room_messages."):]).Call([]reflect.Value{reflect.ValueOf(session), reflect.ValueOf(roomMsg), reflect.ValueOf(bytes), innerMsgValue})
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

func (this *RoomMessageDispatcher) StartListen() *RoomMessageDispatcher  {
	udpListenAddr, _ := net.ResolveUDPAddr("udp", ":2012")
	udpConn, udpListenErr := net.ListenUDP("udp", udpListenAddr)
	if udpListenErr != nil{
		log.Println(udpListenErr)
	}
	log.Println("Listen at udp port 2012 for room message dispatcher")
	this.udpConn = udpConn
	this.roomChannel = make(chan ConnRoomUdpMsg)
	this.wsMsgChan = make(chan WebSocketMsg, 256)
	go PlayerRoomUdpMsgRecvLoop(this.roomChannel, udpConn)
	http.HandleFunc("/room", this.ServeRoomWS)
	return this
}

func PlayerRoomUdpMsgRecvLoop(roomChannel chan ConnRoomUdpMsg, conn *net.UDPConn)  {
	recvBuf := make([]byte, 256)
	for  {
		bcount, remoteAddr, err := conn.ReadFromUDP(recvBuf)
		if err != nil{
			panic(err)
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
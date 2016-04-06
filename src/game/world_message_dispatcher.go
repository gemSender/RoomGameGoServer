package game

import (
	"../messages/world_messages/proto_files"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"io"
	"../utility"
	"github.com/gorilla/websocket"
	"bufio"
	"net/http"
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

type WorldMessageConn interface {
	WriteMessage([]byte) error
	ReceiveMessage() []byte
	Close()
}

type TcpMessageConn struct {
	Conn net.Conn
	SendChan chan []byte
	writer *bufio.Writer
}

func  (this *TcpMessageConn) Close() {
	this.Conn.Close()
}

func (this *TcpMessageConn) WriteMessage(bytes []byte)  error{
	lenBuf := []byte{0, 0, 0, 0}
	log.Println(len(bytes), " bytes will be sent")
	utility.GetBytesFromInt32(int32(len(bytes)), lenBuf)
	this.writer.Write(lenBuf)
	this.writer.Write(bytes)
	this.writer.Flush()
	return nil
}

func (this *TcpMessageConn) ReceiveMessage() []byte {
	lenBuf := []byte{0, 0, 0, 0}
	reader := this.Conn
	receiveNbytes := func(n int32, buf []byte) error{
		sum := int32(0)
		for sum < n {
			readLen, err := reader.Read(buf[sum:n])
			if err != nil{
				return err
			}
			sum += int32(readLen)
		}
		return nil
	}
	tryReceive_exit := func(n int32, buf []byte) (exit bool) {
		err := receiveNbytes(n, buf)
		if err == io.EOF{
			log.Println("client ", reader.RemoteAddr(), " disconnected")
			exit = true
			return
		}else if err != nil{
			log.Println("client ", reader.RemoteAddr(), " disconnected")
			exit = true
			return
		}
		exit = false
		return
	}
	if tryReceive_exit(4, lenBuf){
		return nil
	}
	msg1Len := utility.BytesToInt32(lenBuf)
	log.Println("receive msg len: ", msg1Len)
	msgBytes := make([]byte, msg1Len)
	if tryReceive_exit(msg1Len, msgBytes){
		return nil
	}
	return msgBytes
}

type WebSocketMsgConn  struct{
	Conn *websocket.Conn
}

func  (this *WebSocketMsgConn) Close() {
	this.Conn.Close()
}

func (this *WebSocketMsgConn) WriteMessage(bytes []byte)  error{
	return this.Conn.WriteMessage(websocket.BinaryMessage, bytes)
}

func (this *WebSocketMsgConn) ReceiveMessage() []byte {
	msgType, bytes, readErr := this.Conn.ReadMessage()
	if msgType == websocket.CloseMessage{
		return nil
	}
	if readErr != nil{
		return  nil
	}
	return  bytes
}

func StartPlayerRecvLoop(exitNotifyWorldChan chan string, exitSingalChan chan int, worldMsgChan chan ClientMsg, sendChan chan []byte, conn WorldMessageConn)  {
	defer func() { conn.Close(); exitSingalChan <- 1}()
	msg1Bytes := conn.ReceiveMessage()
	if msg1Bytes == nil{
		return
	}
	idChannel := make(chan string)
	worldMsgChan <- ClientMsg{Bytes:msg1Bytes, SendChannel:sendChan, IdChannel:idChannel}
	playerId := <- idChannel
	defer func() {exitNotifyWorldChan <- playerId}()
	for {
		msgBytes := conn.ReceiveMessage()
		if msgBytes == nil{
			return
		}
		worldMsgChan <- ClientMsg{Bytes:msgBytes, SendChannel:sendChan}
	}
}

func StartPlayerSendLoop(sendChan chan []byte, exitSingalChan chan int, conn WorldMessageConn)  {
	for {
		select {
		case bytes := <- sendChan:
			log.Println(len(bytes), " bytes will be sent")
			conn.WriteMessage(bytes)
		case <-exitSingalChan:
			return
		}
	}
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
	worldMsgChan chan ClientMsg
	exitNotifyWorldChan chan string
}

func CreateWorldMessageDispatcher() *WorldMessageDispatcher{
	ret := &WorldMessageDispatcher{}
	ret.msgActionDict = make(map[world_messages.MessageType]func(WorldSession, *world_messages.WorldMessage, []byte))
	ret.playerChanDict = make(map[string]WorldSession)
	worldMsgDispatcher = ret
	return ret
}

func (this *WorldMessageDispatcher) GetSession(playerId string)  *WorldSession{
	session, exists := this.playerChanDict[playerId]
	if exists{
		return &session
	}
	return nil
}

func (this *WorldMessageDispatcher) RegisterHandler(msgType world_messages.MessageType, action func(WorldSession, *world_messages.WorldMessage, []byte))  {
	this.msgActionDict[msgType] = action
}

func (this *WorldMessageDispatcher) StartRecvLoop()  {
	world := world_instance
	for {
		select {
		case exitPlayerId := <- this.exitNotifyWorldChan:
			world.OnPlayerDisconnect(exitPlayerId)
			delete(this.playerChanDict, exitPlayerId)
		case clientMsg := <- this.worldMsgChan:
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

func (this *WorldMessageDispatcher) StartListenTcp()  {
	tcpListener, listenErr := net.Listen("tcp", ":1234")
	if listenErr != nil{
		log.Panic(listenErr)
	}
	log.Println("Listen at Tcp port 1234 for world message dispatcher")
	for  {
		conn, acError := tcpListener.Accept()
		if acError != nil{
			log.Panic(acError)
		}
		sendChan := make(chan []byte, 128)
		exitSingalChan := make(chan int)
		connInterface := &TcpMessageConn{Conn:conn, writer : bufio.NewWriter(conn)}
		go StartPlayerSendLoop(sendChan, exitSingalChan, connInterface)
		go StartPlayerRecvLoop(this.exitNotifyWorldChan, exitSingalChan, this.worldMsgChan, sendChan, connInterface)
	}
}

func (this *WorldMessageDispatcher) ServeWebsocket(w http.ResponseWriter, r *http.Request) {
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
	sendChan := make(chan []byte, 128)
	exitSingalChan := make(chan int)
	connInterface := &WebSocketMsgConn{Conn:ws}
	go StartPlayerSendLoop(sendChan, exitSingalChan, connInterface)
	go StartPlayerRecvLoop(this.exitNotifyWorldChan, exitSingalChan, this.worldMsgChan, sendChan, connInterface)
}

func (this *WorldMessageDispatcher) StartListenWebSocket()  {
	http.HandleFunc("/world", this.ServeWebsocket)
}

func (this *WorldMessageDispatcher) StartListen() *WorldMessageDispatcher{
	this.worldMsgChan = make(chan ClientMsg, 1024)
	this.exitNotifyWorldChan = make(chan string, 1024)
	go this.StartListenTcp()
	this.StartListenWebSocket()
	return this
}
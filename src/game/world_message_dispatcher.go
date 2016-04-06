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
	"reflect"
	"../messages/proto_files"
)
var worldMsgDispatcher *WorldMessageDispatcher;

type ClientMsg struct{
	SendChannel chan <- []byte
	Bytes []byte
	IdChannel chan <- string
	PlayerId *string
}

type WorldSession struct {
	SendChannel chan <-[]byte
	PlayerId string
	IdChannel chan <- string
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
	log.Println("receive ", len(msg1Bytes), " bytes from client")
	if msg1Bytes == nil{
		return
	}
	idChannel := make(chan string)
	worldMsgChan <- ClientMsg{Bytes:msg1Bytes, SendChannel:sendChan, IdChannel:idChannel}
	playerId := <- idChannel
	defer func() {exitNotifyWorldChan <- playerId}()
	for {
		msgBytes := conn.ReceiveMessage()
		log.Println("receive ", len(msg1Bytes), " bytes from client")
		if msgBytes == nil{
			return
		}
		worldMsgChan <- ClientMsg{Bytes:msgBytes, SendChannel:sendChan, PlayerId:&playerId}
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

func (this *WorldSession) Send(msgId int32, innerMsg proto.Message) {
	msg := messages.GenMessage{}
	msg.MsgId = &msgId
	bytes, encodeErr1 := proto.Marshal(innerMsg)
	if encodeErr1 != nil{
		log.Panic(encodeErr1)
	}
	msg.Buf = bytes
	msgType := proto.MessageName(innerMsg)
	msg.Type = &msgType
	msgBytes, encodeErr2 := proto.Marshal(&msg)
	if encodeErr2 != nil{
		log.Panic(encodeErr2)
	}
	this.SendBytes(msgBytes)
}

func (this *WorldSession) SendBytes(bytes []byte)  {
	this.SendChannel <- bytes
}

type WorldMessageDispatcher struct{
	playerChanDict map[string]WorldSession
	worldMsgChan chan ClientMsg
	exitNotifyWorldChan chan string
}

func CreateWorldMessageDispatcher() *WorldMessageDispatcher{
	ret := &WorldMessageDispatcher{}
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

func (this *WorldMessageDispatcher) RegisterPlayer(session WorldSession)  {
	session.IdChannel <- session.PlayerId
	this.playerChanDict[session.PlayerId] = session
}

func (this *WorldMessageDispatcher) StartRecvLoop()  {
	world := world_instance
	for {
		select {
		case exitPlayerId := <- this.exitNotifyWorldChan:
			world.OnPlayerDisconnect(exitPlayerId)
			delete(this.playerChanDict, exitPlayerId)
		case clientMsg := <- this.worldMsgChan:
			msg := &messages.GenMessage{}
			parseErr := proto.Unmarshal(clientMsg.Bytes, msg)
			if parseErr != nil{
				panic(parseErr)
			}
			var session WorldSession
			if clientMsg.PlayerId == nil{
				session = WorldSession{SendChannel:clientMsg.SendChannel, IdChannel:clientMsg.IdChannel}
			}else{
				session = this.playerChanDict[*clientMsg.PlayerId]
			}
			innerMsgValue := reflect.New(proto.MessageType(msg.GetType()).Elem())
			decodeErr := proto.Unmarshal(msg.GetBuf(), innerMsgValue.Interface().(proto.Message))
			if decodeErr != nil{
				log.Panic(decodeErr)
			}
			method := reflect.ValueOf(world).MethodByName(msg.GetType()[len("world_messages."):])
			method.Call([]reflect.Value{reflect.ValueOf(session), reflect.ValueOf(msg.GetMsgId()), innerMsgValue})
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

func (this *WorldMessageDispatcher) PushMutiply(innerMsg proto.Message, match func(string) bool)  {
	replyMsg := &messages.GenMessage{}
	msgId := int32(-1)
	replyMsg.MsgId = &msgId
	msgType := proto.MessageName(innerMsg)
	replyMsg.Type = &msgType
	innerBytes, encodeErr := proto.Marshal(innerMsg)
	if encodeErr != nil{
		log.Panic(encodeErr)
	}
	replyMsg.Buf = innerBytes
	bytes, encodeErr1 := proto.Marshal(replyMsg)
	if encodeErr1 != nil{
		log.Panic(encodeErr1)
	}
	for p, sess := range this.playerChanDict{
		if match(p){
			sess.SendBytes(bytes)
		}
	}
}
package main

import (
	"net"
	"log"
	"io"
	"bufio"
	"./utility"
	"./game"
)

func PlayerRecvLoop(exitNotifyWorldChan chan string, exitSingalChan chan int, worldMsgChan chan game.ClientMsg, sendChan chan []byte, reader net.Conn)  {
	exitFun := func() {
		reader.Close()
		exitSingalChan <- 1
	}
	defer exitFun()
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
	lenBuf := []byte{0, 0, 0, 0}
	if tryReceive_exit(4, lenBuf){
		return
	}
	msg1Len := utility.BytesToInt32(lenBuf)
	log.Println("receive msg len: ", msg1Len)
	msg1Bytes := make([]byte, msg1Len)
	if tryReceive_exit(msg1Len, msg1Bytes){
		return
	}
	idChannel := make(chan string)
	{
		worldMsgChan <- game.ClientMsg{Bytes:msg1Bytes, SendChannel:sendChan, IdChannel:idChannel}
		playerId := <- idChannel
		defer func() {exitNotifyWorldChan <- playerId}()
	}
	for {
		if tryReceive_exit(4, lenBuf){
			return
		}
		msgLen := utility.BytesToInt32(lenBuf)
		msgBytes := make([]byte, msgLen)
		if tryReceive_exit(msgLen, msgBytes){
			return
		}
		worldMsgChan <- game.ClientMsg{Bytes:msgBytes, SendChannel:sendChan}
	}
}

func PlayerSendLoop(sendChan chan []byte, exitSingalChan chan int, conn net.Conn)  {
	writer := bufio.NewWriter(conn)
	lenBuf := []byte{0, 0, 0, 0}
	for {
		select {
		case bytes := <- sendChan:
			log.Println(len(bytes), " bytes will be sent")
			utility.GetBytesFromInt32(int32(len(bytes)), lenBuf)
			writer.Write(lenBuf)
			writer.Write(bytes)
			writer.Flush()
		case <-exitSingalChan:
			return
		}
	}
}

func PlayerRoomMsgRecvLoop(roomChannel chan game.ConnRoomMsg, conn *net.UDPConn)  {
	recvBuf := make([]byte, 256)
	for  {
		bcount, remoteAddr, error := conn.ReadFromUDP(recvBuf)
		if error != nil{
			panic(error)
		}
		copiedBytes := make([]byte, bcount)
		copy(copiedBytes, recvBuf[:bcount])
		msg := game.ConnRoomMsg{RemoteAddr:remoteAddr, Bytes:copiedBytes}
		roomChannel <- msg
	}
}

func main(){
	tcpListener, listenErr := net.Listen("tcp", ":1234")
	if listenErr != nil{
		log.Panic(listenErr)
	}
	worldMsgChan := make(chan game.ClientMsg, 1024)
	exitNotifyWorldChan := make(chan string, 1024)
	roomChannel := make(chan game.ConnRoomMsg)
	dispatcher := game.CreateWorldMessageDispatcher()
	game.CreateWorld()
	go dispatcher.StartRecvLoop(exitNotifyWorldChan, worldMsgChan)
	udpListenAddr, _ := net.ResolveUDPAddr("udp", ":2012")
	udpConn, udpListenErr := net.ListenUDP("udp", udpListenAddr)
	if udpListenErr != nil{
		log.Println(udpListenErr)
	}
	go PlayerRoomMsgRecvLoop(roomChannel, udpConn)
	roomMsgDispatcher := game.CreateRoomMsgDispatcher()
	go roomMsgDispatcher.StartReceiveLoop(roomChannel, udpConn)
	for  {
		conn, acError := tcpListener.Accept()
		if acError != nil{
			log.Panic(acError)
		}
		sendChan := make(chan []byte, 128)
		exitSingalChan := make(chan int)
		go PlayerSendLoop(sendChan, exitSingalChan, conn)
		go PlayerRecvLoop(exitNotifyWorldChan, exitSingalChan, worldMsgChan, sendChan, conn)
	}
}

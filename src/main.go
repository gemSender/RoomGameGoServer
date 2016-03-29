package main

import (
	"net"
	"log"
	"io"
)

func PlayerRecvLoop(exitSingalChan chan int, reader io.Reader)  {

}

func PlayerSendLoop(sendChan chan []byte, exitSingalChan chan int, writer io.Writer)  {
	for {
		select {
		case bytes := <- sendChan:
			writer.Write(bytes)
		case <-exitSingalChan:
			return
		}
	}
}

func main(){
	tcpListener, listenErr := net.Listen("tcp", ":1234")
	if listenErr != nil{
		log.Panic(listenErr)
	}
	for  {
		conn, acError := tcpListener.Accept()
		if acError != nil{
			log.Panic(acError)
		}
		sendChan := make(chan []byte, 128)
		exitSingalChan := make(chan int)
		go PlayerSendLoop(sendChan, exitSingalChan, conn)
		go PlayerRecvLoop(exitSingalChan, conn)
	}
}

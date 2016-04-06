package main

import (
	"log"
	"./game"
	"flag"
	"net/http"
)


func main(){
	worldMsgDispatcher := game.CreateWorldMessageDispatcher()
	game.CreateWorld()
	go worldMsgDispatcher.StartListen().StartRecvLoop()
	roomMsgDispatcher := game.CreateRoomMsgDispatcher()
	go roomMsgDispatcher.StartListen().StartReceiveLoop()
	var addr = flag.String("addr", ":8080", "http service address")
	log.Println("Listen at port 8080 for http service")
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

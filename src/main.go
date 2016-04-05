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
	go worldMsgDispatcher.StartRecvLoop()
	roomMsgDispatcher := game.CreateRoomMsgDispatcher()
	go roomMsgDispatcher.StartReceiveLoop()
	var addr = flag.String("addr", ":8080", "http service address")
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

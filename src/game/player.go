package game

const maxCommandCount int = 16

type Player struct {
	Id string
	Index int
	nextMsgId int
	lastFrame int
}
package game

type Room struct {
	Id string
	PlayerCount int
	Capacity int
	Players []*Player
}

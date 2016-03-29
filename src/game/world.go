package game

var world_instance *World

type World struct {
	Rooms []*Room
	PlayerId_Index_Map map[string]int
	PlayerIndex_Room_Map map[int]*Room
}

func CreateWorld()  *World{
	ret := &World{}
	ret.Rooms = make([]*Room, 32)
	ret.PlayerId_Index_Map = make(map[string]int)
	ret.PlayerIndex_Room_Map = make(map[int]*Room)
	return ret
}

func (this *World) ()  {
	
}
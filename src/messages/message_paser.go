package messages

type MessagePaser interface{
	Parse() interface{}
	Decode(interface{})
}
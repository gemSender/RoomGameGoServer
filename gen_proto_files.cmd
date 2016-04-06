protoc --go_out=src/messages/ proto_files/Messages.proto
protoc --go_out=src/messages/room_messages proto_files/RoomMessage.proto
protoc --go_out=src/messages/world_messages proto_files/WorldMessage.proto
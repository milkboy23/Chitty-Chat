package main

import (
	proto "Chitty-Chat/API"
	"log"
	"net"

	"google.golang.org/grpc"
)

type ChatServer struct {
	proto.UnimplementedChatServerServer
}

func main() {

}

func (s *ChatServer) startServer() {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	proto.RegisterChatServerServer(grpcServer, s)

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

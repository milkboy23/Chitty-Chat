package main

import (
	proto "Chitty-Chat/GRPC"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
)

type ChatServer struct {
	proto.UnimplementedChatServiceServer
	clients     map[string]*Client
	lamportTime int64
}

type Client struct {
	username string
	stream   proto.ChatService_JoinChatServer
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients:     make(map[string]*Client),
		lamportTime: 0,
	}
}

func main() {
	server := NewChatServer()
	server.StartServer()
}

func (s *ChatServer) StartServer() {
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatalf("Failed to listen on port 5050: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcServer, s)
	log.Printf("ChatServer is running.")

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func (s *ChatServer) JoinChat(user *proto.UserRequest, stream proto.ChatService_JoinChatServer) error {
	if _, exists := s.clients[user.Username]; exists {
		log.Printf("User %s has already joined.", user.Username)
	} else {
		s.clients[user.Username] = &Client{stream: stream}

		joinMessage := fmt.Sprintf("User %s has joined the chat.", user.Username)
		log.Printf(joinMessage)

		joinMsg := &proto.Chat{
			Username:  "Server",
			Message:   joinMessage,
			Timestamp: 0,
		}
		s.broadcastMessage(joinMsg)
	}

	return nil
}

func (s *ChatServer) BroadcastMessage(ctx context.Context, chat *proto.Chat) (*proto.Empty, error) {
	log.Printf("%d | %s: %s", chat.Timestamp, chat.Username, chat.Message)
	s.broadcastMessage(chat)

	return &proto.Empty{}, nil
}

func (s *ChatServer) LeaveChat(ctx context.Context, user *proto.UserRequest) (*proto.Empty, error) {
	delete(s.clients, user.Username)

	leaveMessage := fmt.Sprintf("User %s left the chat.", user.Username)
	log.Printf(leaveMessage)

	leaveMsg := &proto.Chat{
		Username:  "Server",
		Message:   leaveMessage,
		Timestamp: 0,
	}
	s.broadcastMessage(leaveMsg)

	return &proto.Empty{}, nil
}

func (s *ChatServer) broadcastMessage(message *proto.Chat) {
	for username, connection := range s.clients {
		if username != message.Username {
			if err := connection.stream.Send(message); err != nil {
				log.Printf("Failed to send message to %s | %v", username, err)
			}
		}
	}
}

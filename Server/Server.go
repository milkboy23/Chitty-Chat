package main

import (
	proto "Chitty-Chat/GRPC"
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
)

const port = 5050

type ChatServer struct {
	proto.UnimplementedChatServiceServer
	clients     map[string]*Client
	lamportTime int64
}

type Client struct {
	username string
	stream   proto.ChatService_JoinChatServer
}

func main() {
	server := NewChatServer()
	server.StartServer()
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients:     make(map[string]*Client),
		lamportTime: -1,
	}
}

func (s *ChatServer) StartServer() {
	portString := fmt.Sprintf(":%d", port)
	listener, listenErr := net.Listen("tcp", portString)
	if listenErr != nil {
		log.Fatalf("Failed to listen on port %s | %v", portString, listenErr)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcServer, s)
	log.Print("ChatService server has started")

	serveListenerErr := grpcServer.Serve(listener)
	if serveListenerErr != nil {
		log.Fatalf("Failed to serve listener | %v", serveListenerErr)
	}
}

func (s *ChatServer) JoinChat(user *proto.UserRequest, stream proto.ChatService_JoinChatServer) error {
	_, userAlreadyJoined := s.clients[user.Username]
	log.Printf("User %s is joining", user.Username)
	if userAlreadyJoined {
		log.Printf("User %s has already joined, ignoring...", user.Username)
		return nil
	}

	newUserClient := &Client{stream: stream}
	s.clients[user.Username] = newUserClient

	joinMessage := fmt.Sprintf("User %s has joined the chat", user.Username)
	log.Print(joinMessage)

	joinMsg := &proto.Chat{
		Username:  "Server",
		Message:   joinMessage,
		Timestamp: -1,
	}
	s.broadcastMessage(joinMsg)
	select {
	case <-stream.Context().Done():
		s.leaveChat(user.Username)
		return status.Error(codes.Canceled, "Stream was closed")
	}
}

func (s *ChatServer) BroadcastMessage(ctx context.Context, chat *proto.Chat) (*proto.Empty, error) {
	s.broadcastMessage(chat)

	return &proto.Empty{}, nil
}

func (s *ChatServer) LeaveChat(ctx context.Context, user *proto.UserRequest) (*proto.Empty, error) {
	s.leaveChat(user.Username)
	return &proto.Empty{}, nil
}

func (s *ChatServer) leaveChat(username string) {
	delete(s.clients, username)
	// TEST IF CLIENT HAS ACTUALLY BEEN DELETED

	leaveMessage := fmt.Sprintf("User %s left the chat", username)
	log.Print(leaveMessage)

	leaveMsg := &proto.Chat{
		Username:  "Server",
		Message:   leaveMessage,
		Timestamp: -1,
	}
	s.broadcastMessage(leaveMsg)
}

func (s *ChatServer) broadcastMessage(message *proto.Chat) {
	log.Printf("Broadcasting message: '%d | %s: %s'", message.Timestamp, message.Username, message.Message)
	for username, userConnection := range s.clients {
		log.Printf("Sending to %s", username)

		sendErr := userConnection.stream.Send(message)
		if errors.Is(sendErr, context.Canceled) {
			log.Printf("Context canceled | %v", sendErr)
			continue
		}

		if sendErr != nil {
			log.Printf("Failed to send message to %s | %v", username, sendErr)
			continue
		}
	}
}

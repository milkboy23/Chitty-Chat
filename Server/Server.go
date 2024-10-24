package main

import (
	proto "Chitty-Chat/GRPC"
	"context"
	"fmt"
	"google.golang.org/grpc"
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
	log.Print("ChatService server registered")

	serveListenerErr := grpcServer.Serve(listener)
	if serveListenerErr != nil {
		log.Fatalf("Failed to serve listener | %v", serveListenerErr)
	}
	log.Print("ChatService server has started")
}

func (s *ChatServer) JoinChat(user *proto.UserRequest, stream proto.ChatService_JoinChatServer) error {
	_, userAlreadyJoined := s.clients[user.Username]
	if userAlreadyJoined {
		log.Printf("User %s has already joined", user.Username)
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

	return nil
}

func (s *ChatServer) BroadcastMessage(ctx context.Context, chat *proto.Chat) (*proto.Empty, error) {
	s.broadcastMessage(chat)

	return &proto.Empty{}, nil
}

func (s *ChatServer) LeaveChat(ctx context.Context, user *proto.UserRequest) (*proto.Empty, error) {
	delete(s.clients, user.Username)
	// TEST IF CLIENT HAS ACTUALLY BEEN DELETED

	leaveMessage := fmt.Sprintf("User %s left the chat", user.Username)
	log.Print(leaveMessage)

	leaveMsg := &proto.Chat{
		Username:  "Server",
		Message:   leaveMessage,
		Timestamp: -1,
	}
	s.broadcastMessage(leaveMsg)

	return &proto.Empty{}, nil
}

func (s *ChatServer) broadcastMessage(message *proto.Chat) {
	log.Printf("Broadcasting message: '%d | %s: %s'", message.Timestamp, message.Username, message.Message)
	for username, userConnection := range s.clients {
		log.Printf("Sending to %s", username)

		sendErr := userConnection.stream.Send(message)
		if sendErr != nil {
			log.Printf("Failed to send message to %s | %v", username, sendErr)
			continue
		}
		log.Printf("Sent message to %s", username)
	}
}

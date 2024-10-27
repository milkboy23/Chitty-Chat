package main

import (
	proto "Chitty-Chat/GRPC"
	"context"
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

func (server *ChatServer) StartServer() {
	portString := fmt.Sprintf(":%d", port)
	listener, listenErr := net.Listen("tcp", portString)
	if listenErr != nil {
		log.Fatalf("Failed to listen on port %s | %v", portString, listenErr)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcServer, server)
	log.Print("ChatService server has started")

	serveListenerErr := grpcServer.Serve(listener)
	if serveListenerErr != nil {
		log.Fatalf("Failed to serve listener | %v", serveListenerErr)
	}
}

func (server *ChatServer) JoinChat(user *proto.UserRequest, stream proto.ChatService_JoinChatServer) error {
	_, userAlreadyJoined := server.clients[user.Username]
	if userAlreadyJoined {
		log.Printf("User %s has already joined, but is requesting to join again, ignoring...", user.Username)
		return nil
	}

	newUserClient := &Client{username: user.Username, stream: stream}
	server.clients[user.Username] = newUserClient

	joinMessage := fmt.Sprintf("User %s has joined the chat", user.Username)
	log.Print(joinMessage)

	joinMsg := &proto.Chat{
		Username:  "Server",
		Message:   joinMessage,
		Timestamp: -1,
	}
	server.broadcastMessage(joinMsg)
	select {
	case <-stream.Context().Done():
		server.LeaveChat(user.Username)
		return status.Error(codes.Canceled, "Stream was closed")
	}
}

func (server *ChatServer) BroadcastMessage(ctx context.Context, chat *proto.Chat) (*proto.Empty, error) {
	server.broadcastMessage(chat)

	return &proto.Empty{}, nil
}

func (server *ChatServer) LeaveChat(username string) {
	delete(server.clients, username)

	leaveMessage := fmt.Sprintf("User %s left the chat", username)
	log.Print(leaveMessage)

	leaveMsg := &proto.Chat{
		Username:  "Server",
		Message:   leaveMessage,
		Timestamp: -1,
	}
	server.broadcastMessage(leaveMsg)
}

func (server *ChatServer) broadcastMessage(message *proto.Chat) {
	log.Printf("Broadcasting message: '%d | %s: %s'", message.Timestamp, message.Username, message.Message)
	for username, userConnection := range server.clients {
		sendErr := userConnection.stream.Send(message)
		if sendErr != nil {
			log.Printf("Failed to send message to %s | %v", username, sendErr)
			continue
		}
	}
}

package main

import (
	proto "Chitty-Chat/GRPC"
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const port = 5050

type ChatServer struct {
	proto.UnimplementedChatServiceServer
	clients     map[string]*Client
	lamportTime int32
}

type Client struct {
	username  string
	stream    proto.ChatService_JoinChatServer
	Timestamp int32
}

func main() {
	server := NewChatServer()
	server.StartServer()
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients:     make(map[string]*Client),
		lamportTime: 0,
	}
}

func (server *ChatServer) StartServer() {
	server.lamportTime++
	portString := fmt.Sprintf(":%d", port)
	listener, listenErr := net.Listen("tcp", portString)
	if listenErr != nil {
		log.Fatalf("Failed to listen on port %s | %v", portString, listenErr)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcServer, server)
	log.Print("ChatService server has started at time %d", server.lamportTime)

	serveListenerErr := grpcServer.Serve(listener)
	if serveListenerErr != nil {
		log.Fatalf("Failed to serve listener | %v", serveListenerErr)
	}
}

func (server *ChatServer) JoinChat(user *proto.UserRequest, stream proto.ChatService_JoinChatServer) error {
	_, userAlreadyJoined := server.clients[user.Username]
	log.Printf("User %s is joining", user.Username)
	if userAlreadyJoined {
		log.Printf("User %s has already joined, ignoring...", user.Username)
		return nil
	}

	maxTimestamp := max(user.Timestamp, server.lamportTime)
	server.lamportTime = maxTimestamp + 1

	newUserClient := &Client{username: user.Username, stream: stream}
	server.clients[user.Username] = newUserClient

	joinMessage := fmt.Sprintf("User %s has joined the chat at Lamport time %d", user.Username, server.lamportTime)
	log.Print(joinMessage)

	joinMsg := &proto.Chat{
		Username:  "Server",
		Message:   joinMessage,
		Timestamp: server.lamportTime,
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

	user := server.clients[username]

	maxTimestamp := max(user.Timestamp, server.lamportTime)
	server.lamportTime = maxTimestamp + 1

	delete(server.clients, username)

	leaveMessage := fmt.Sprintf("User %s left the chat at Lamport time %d", username, server.lamportTime)
	log.Print(leaveMessage)

	leaveMsg := &proto.Chat{
		Username:  "Server",
		Message:   leaveMessage,
		Timestamp: server.lamportTime,
	}
	server.broadcastMessage(leaveMsg)
}

func (server *ChatServer) broadcastMessage(message *proto.Chat) {
	log.Printf("Broadcasting message: '%d | %s: %s'", message.Timestamp, message.Username, message.Message)
	for username, userConnection := range server.clients {
		log.Printf("Sending to %s", username)

		sendErr := userConnection.stream.Send(message)
		if sendErr != nil {
			log.Printf("Failed to send message to %s | %v", username, sendErr)
			continue
		}
	}
}

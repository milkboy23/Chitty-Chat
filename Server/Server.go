package main

import (
	proto "Chitty-Chat/GRPC"
	"context"
	"fmt"
	"google.golang.org/grpc/metadata"
	"log"
	"net"
	"strconv"

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
		lamportTime: 0,
	}
}

func (server *ChatServer) updateTimestamp(incomingTimestamp int32) {
	server.lamportTime = max(incomingTimestamp, server.lamportTime) + 1
}

func (server *ChatServer) StartServer() {
	portString := fmt.Sprintf(":%d", port)
	listener, listenErr := net.Listen("tcp", portString)
	if listenErr != nil {
		log.Fatalf("Failed to listen on port %s | %v", portString, listenErr)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcServer, server)
	log.Printf("LT%d | ChatService server has started", server.lamportTime)

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

	md := metadata.Pairs("lamport-timestamp", strconv.Itoa(int(server.lamportTime)))
	streamHeaderErr := stream.SetHeader(md)
	if streamHeaderErr != nil {
		log.Fatalf("Failed to set header on stream | %v", streamHeaderErr)
	}

	newUserClient := &Client{username: user.Username, stream: stream}
	server.clients[user.Username] = newUserClient

	server.lamportTime++
	server.updateTimestamp(user.Timestamp)

	joinMessage := fmt.Sprintf("User %s join request received at LT%d", user.Username, server.lamportTime)
	log.Print(joinMessage)

	joinMsg := &proto.Chat{
		Username:  "Server",
		Message:   joinMessage,
		Timestamp: server.lamportTime,
	}
	server.broadcastMessage(joinMsg)

	select {
	case <-stream.Context().Done():
		user.Timestamp = server.lamportTime
		server.leaveChat(user)
		return status.Error(codes.Canceled, "Stream was closed")
	}
}

func (server *ChatServer) BroadcastMessage(ctx context.Context, chat *proto.Chat) (*proto.Empty, error) {
	server.updateTimestamp(chat.Timestamp)
	log.Printf("LT%d | Message received", server.lamportTime)
	server.broadcastMessage(chat)

	return &proto.Empty{}, nil
}

func (server *ChatServer) leaveChat(user *proto.UserRequest) {
	server.updateTimestamp(user.Timestamp)
	delete(server.clients, user.Username)

	leaveMessage := fmt.Sprintf("User %s leave request received at LT%d", user.Username, server.lamportTime)
	log.Print(leaveMessage)

	leaveMsg := &proto.Chat{
		Username:  "Server",
		Message:   leaveMessage,
		Timestamp: server.lamportTime,
	}
	server.broadcastMessage(leaveMsg)
}

func (server *ChatServer) broadcastMessage(message *proto.Chat) {
	server.lamportTime++
	message.Timestamp = server.lamportTime
	log.Printf("LT%d | Broadcasting: '%s: %s'", server.lamportTime, message.Username, message.Message)
	for username, userConnection := range server.clients {
		sendErr := userConnection.stream.Send(message)
		if sendErr != nil {
			log.Printf("Failed to send message to %s | %v", username, sendErr)
			continue
		}
	}
}

package main

import (
	proto "Chitty-Chat/GRPC"
	"bufio"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
	"strings"
)

const port = 5050

var reader *bufio.Scanner
var username string
var programFinished = make(chan bool)

func main() {
	getUsername()

	clientConnection, client := startClient()
	chatStream := joinChat(client)

	go listenToStream(chatStream)
	go listenForInput(client)

	<-programFinished

	closeClient(clientConnection)
}

func getUsername() {
	log.Print("Please enter a username:")
	reader = bufio.NewScanner(os.Stdin)
	reader.Scan()
	username = reader.Text()
}

func startClient() (*grpc.ClientConn, proto.ChatServiceClient) {
	portString := fmt.Sprintf(":%d", port)
	dialOptions := grpc.WithTransportCredentials(insecure.NewCredentials())
	connection, connectionEstablishErr := grpc.NewClient(portString, dialOptions)
	if connectionEstablishErr != nil {
		log.Fatalf("Could not establish connection on port %s | %v", portString, connectionEstablishErr)
	}

	log.Printf("Client has been started on port %s", portString)

	return connection, proto.NewChatServiceClient(connection)
}

func closeClient(connection *grpc.ClientConn) {
	connectionCloseErr := connection.Close()
	if connectionCloseErr != nil {
		log.Fatalf("Could not close connection | %v", connectionCloseErr)
	}
	log.Print("Client connection has been closed")
}

func joinChat(client proto.ChatServiceClient) proto.ChatService_JoinChatClient {
	user := proto.UserRequest{Username: username, Timestamp: -1}
	chatStream, joinErr := client.JoinChat(context.Background(), &user)
	if joinErr != nil {
		log.Fatalf("Could not join chat | %v", joinErr)
	}
	log.Printf("Chat successfully joined as %s!", user.Username)

	return chatStream
}

func listenToStream(stream proto.ChatService_JoinChatClient) {
	for {
		message, chatStreamErr := stream.Recv()
		if chatStreamErr == io.EOF { // This is still broken and doesn't make any sense
			log.Printf("Server closed the stream")
			//programFinished <- true
			return
		}
		if chatStreamErr != nil {
			log.Fatalf("Error receiving message | %v", chatStreamErr)
		}
		log.Printf("%d | %s: %s", message.Timestamp, message.Username, message.Message)
	}
}

func listenForInput(client proto.ChatServiceClient) {
	for {
		reader.Scan()
		userInput := reader.Text()

		if len(userInput) <= 0 {
			log.Print("Input was empty")
			continue
		}

		if strings.ToLower(userInput) == "leave" {
			leaveChat(client)
			programFinished <- true
			return
		}

		broadcastMessage(client, userInput)
	}
}

func leaveChat(client proto.ChatServiceClient) {
	user := &proto.UserRequest{Username: username, Timestamp: -1}
	_, leaveErr := client.LeaveChat(context.Background(), user)
	if leaveErr != nil {
		log.Fatalf("Could not leave chat | %v", leaveErr)
	}
	log.Print("Successfully left chat")
}

func broadcastMessage(client proto.ChatServiceClient, userInput string) {
	message := &proto.Chat{Username: username, Message: userInput, Timestamp: -1}
	log.Printf("%d | %s: %s", message.Timestamp, message.Username, message.Message)

	_, broadcastErr := client.BroadcastMessage(context.Background(), message)
	if broadcastErr != nil {
		log.Fatalf("Error Broadcasting Message | %v", broadcastErr)
	}
	log.Print("Successfully broadcast message")
}

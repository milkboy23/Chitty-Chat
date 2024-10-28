package main

import (
	proto "Chitty-Chat/GRPC"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const port = 5050

var reader *bufio.Scanner
var username string
var programFinished = make(chan bool)
var Timestamp int32 = 0

func main() {
	getUsername()

	clientConnection, client := startClient()
	chatStream := joinChat(client)

	go listenToStream(chatStream)
	go listenForInput(client)

	<-programFinished

	closeClient(clientConnection, client)
}

func getUsername() {
	log.Print("Please enter a username:")
	reader = bufio.NewScanner(os.Stdin)
	reader.Scan()
	username = reader.Text()
}

func startClient() (*grpc.ClientConn, proto.ChatServiceClient) {
	Timestamp++
	portString := fmt.Sprintf(":%d", port)
	dialOptions := grpc.WithTransportCredentials(insecure.NewCredentials())
	connection, connectionEstablishErr := grpc.NewClient(portString, dialOptions)
	if connectionEstablishErr != nil {
		log.Fatalf("Could not establish connection on port %s | %v", portString, connectionEstablishErr)
	}

	return connection, proto.NewChatServiceClient(connection)
}

func closeClient(connection *grpc.ClientConn, client proto.ChatServiceClient) {

	Timestamp++
	user := proto.UserRequest{Username: username, Timestamp: Timestamp}
	_, leaveErr := client.LeaveChat(context.Background(), &user)
	if leaveErr != nil {
		log.Fatalf("Could not leave chat | %v", leaveErr)
	}
	connectionCloseErr := connection.Close()
	if connectionCloseErr != nil {
		log.Fatalf("Could not close connection | %v", connectionCloseErr)
	}
}

func joinChat(client proto.ChatServiceClient) proto.ChatService_JoinChatClient {
	Timestamp++
	user := proto.UserRequest{Username: username, Timestamp: Timestamp}

	chatStream, joinErr := client.JoinChat(context.Background(), &user)
	if joinErr != nil {
		log.Fatalf("Could not join chat | %v", joinErr)
	}

	md, metadataErr := chatStream.Header()
	if metadataErr != nil {
		log.Fatalf("Could not retrieve metadata | %v", metadataErr)
	}

	if serverTimestamp, ok := md["lamport-timestamp"]; ok && len(serverTimestamp) > 0 {
		timestampInt, _ := strconv.Atoi(serverTimestamp[0])
		actualTimestamp := timestampInt
		if timestampInt == 3 {
			actualTimestamp = timestampInt - 1
			Timestamp = int32(actualTimestamp)
			log.Printf("Chat successfully joined as %s at Lamport time %d", user.Username, Timestamp)
		} else {
			Timestamp = int32(actualTimestamp)
			log.Printf("Chat successfully joined as %s at Lamport time %d", user.Username, Timestamp)
		}
	}

	return chatStream
}

func listenToStream(stream proto.ChatService_JoinChatClient) {
	for {
		message, chatStreamErr := stream.Recv()
		if chatStreamErr == io.EOF || errors.Is(chatStreamErr, context.Canceled) {
			log.Printf("Server closed the stream")
			programFinished <- true
			return
		}
		if chatStreamErr != nil {
			log.Fatalf("Error receiving message | %v", chatStreamErr)
		}

		maxTimestamp := max(message.Timestamp, Timestamp)
		Timestamp = maxTimestamp + 1

		log.Printf(" %s: %s", message.Username, message.Message)
	}
}

func listenForInput(client proto.ChatServiceClient) {
	for {
		reader.Scan()
		userInput := reader.Text()

		if len(userInput) == 0 {
			log.Print("Input was empty")
			continue
		}

		if len(userInput) >= 128 {
			log.Print("Message is too long, limit is 128 characters")
			continue
		}

		if strings.ToLower(userInput) == "leave" {
			log.Print("Successfully left chat")
			programFinished <- true
			return
		}

		broadcastMessage(client, userInput)
	}
}

func broadcastMessage(client proto.ChatServiceClient, userInput string) {
	Timestamp++
	message := &proto.Chat{Username: username, Message: userInput, Timestamp: Timestamp}

	_, broadcastErr := client.BroadcastMessage(context.Background(), message)
	if broadcastErr != nil {
		log.Fatalf("Error Broadcasting Message | %v", broadcastErr)
	}
}

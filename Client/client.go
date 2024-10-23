package main

import (
	proto "Chitty-Chat/GRPC"
	"bufio"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"os"
)

var username string
var reader *bufio.Scanner
var done = make(chan bool)

func main() {
	log.Print("Please enter a username:")
	reader = bufio.NewScanner(os.Stdin)
	reader.Scan()
	username = reader.Text()

	client := StartClient()
	stream := joinChat(client)

	go listenToStream(stream)
	go listenForInput(client)

	<-done
}

func StartClient() proto.ChatServiceClient {
	conn, err := grpc.NewClient(":5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}

	return proto.NewChatServiceClient(conn)
}

func joinChat(client proto.ChatServiceClient) proto.ChatService_JoinChatClient {
	user := proto.User{Username: username, Timestamp: 0}

	stream, err := client.JoinChat(context.Background(), &user)
	if err != nil {
		log.Fatalf("Could not join chat: %v", err)
	}
	log.Printf("Chat successfully joined as %s!", user.Username)

	return stream
}

func listenToStream(stream proto.ChatService_JoinChatClient) {
	for {
		message, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Server closed the stream")
			//done <- true
			return
		}
		if err != nil {
			log.Fatalf("Error receiving message: %v", err)
		}
		log.Printf("%d | %s: %s", message.Timestamp, message.Username, message.Message)
	}
}

func listenForInput(client proto.ChatServiceClient) {
	for {
		reader.Scan()
		input := reader.Text()
		if len(input) > 0 {
			if input == "leave" {
				_, err := client.LeaveChat(context.Background(), &proto.User{Username: username})
				if err != nil {
					log.Fatalf("Could not leave chat: %v", err)
				}
				continue
			}

			message := &proto.Chat{Username: username, Message: input, Timestamp: 0}
			log.Printf("%d | %s: %s", message.Timestamp, message.Username, message.Message)
			_, err := client.BroadcastMessages(context.Background(), message)
			if err != nil {
				log.Fatalf("Error Broadcasting Messages: %v", err)
			}
		}
	}
}

package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"

	pb "github.com/hyperledger/fabric-private-chaincode/tle_go/tlegrpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// Set up a connection to the server
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a new gRPC client
	client := pb.NewTleServiceClient(conn)

	// Prepare the request
	request := &pb.MetaRequest{
		Namespace: "example",
		Key:       "mykey",
	}

	// Send the gRPC request
	response, err := client.GetMeta(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to call GetMeta: %v", err)
	}

	// Process the response
	data := response.GetData()
	lastCommitHash := response.GetLastCommitHash()
	fmt.Printf("Calling GetMeta, Received data: %s, lastCommitHash: %s\n", string(data), string(lastCommitHash))

	request2 := &pb.Empty{}
	response, err = client.GetSession(context.Background(), request2)
	if err != nil {
		log.Fatalf("Failed to call GetSession: %v", err)
	}
	data = response.GetData()
	lastCommitHash = response.GetLastCommitHash()
	fmt.Printf("Calling GetSession, Received data: %s, lastCommitHash: %s\n", string(data), string(lastCommitHash))
}

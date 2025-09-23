package main

import (
	"context"
	"log"
	"time"

	pb "github.com/go-example/gRPC/timeout/pb"
	wrapper "github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

const (
	address = "127.0.0.1:50051"
)

func main() {
	// Setting up a connection to the server.
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn)
	clientDeadline := time.Now().Add(time.Duration(7 * time.Second))
	ctx, cancel := context.WithDeadline(context.Background(), clientDeadline)
	defer cancel()

	// Add Order
	order1 := pb.Order{Id: "101", Items: []string{"iPhone XS", "Mac Book Pro"}, Destination: "San Jose, CA", Price: 2300.00}
	res, _ := client.AddOrder(ctx, &order1)
	if res != nil {
		log.Print("AddOrder Response -> ", res.Value)
	}

	// Get Order
	retrievedOrder, err := client.GetOrder(ctx, &wrapper.StringValue{Value: "106"})
	log.Print("GetOrder Response -> : ", retrievedOrder, err)
}

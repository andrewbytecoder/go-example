package main

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"log"
	"net"
	/*"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"*/
	pb "github.com/go-example/gRPC/timeout/pb"

	wrapper "github.com/golang/protobuf/ptypes/wrappers"
)

const (
	orderBatchSize = 3
)

var orderMap = make(map[string]pb.Order)

type server struct {
	pb.UnimplementedOrderManagementServer
	orderMap map[string]*pb.Order
}

// AddOrder Simple RPC
func (s *server) AddOrder(ctx context.Context, orderReq *pb.Order) (*wrapper.StringValue, error) {
	log.Printf("Order Added. ID : %v", orderReq.Id)
	orderMap[orderReq.Id] = *orderReq
	sleepDuration := 5
	log.Printf("Sleeping for %v seconds", sleepDuration)

	time.Sleep(time.Duration(sleepDuration) * time.Second)

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		log.Printf("Deadline exceeded for order %v", orderReq.Id)
		return nil, status.Errorf(codes.DeadlineExceeded, "Deadline exceeded for order %v", orderReq.Id)
	}

	return &wrapper.StringValue{Value: "Order Added: " + orderReq.Id}, nil
}

// GetOrder Simple RPC
func (s *server) GetOrder(ctx context.Context, orderId *wrapper.StringValue) (*pb.Order, error) {
	ord, exists := orderMap[orderId.Value]
	if exists {
		return &ord, status.New(codes.OK, "").Err()
	}

	return nil, status.Errorf(codes.NotFound, "Order does not exist. : %s", orderId.GetValue())
}

func main() {
	initSampleData()
	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterOrderManagementServer(s, &server{})
	// Register reflection service on gRPC server.
	// reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func initSampleData() {
	orderMap["102"] = pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	orderMap["103"] = pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	orderMap["104"] = pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	orderMap["105"] = pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	orderMap["106"] = pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
}

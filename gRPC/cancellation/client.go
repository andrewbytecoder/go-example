package main

import (
	"context"
	"io"
	"log"
	"time"

	pb "github.com/go-example/gRPC/cancellation/pb"
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

	// Process Order
	streamProcOrder, _ := client.ProcessOrders(ctx)
	_ = streamProcOrder.Send(&wrapper.StringValue{Value: "102"})
	_ = streamProcOrder.Send(&wrapper.StringValue{Value: "103"})
	_ = streamProcOrder.Send(&wrapper.StringValue{Value: "104"})

	channel := make(chan struct{})

	go asncClientBidirectionalRPC(streamProcOrder, channel)
	time.Sleep(time.Millisecond * 1000)

	// Cancelling the RPC
	cancel()
	log.Printf("Cancelling RPC..., RPC status : %s", ctx.Err())

	_ = streamProcOrder.Send(&wrapper.StringValue{Value: "101"})
	_ = streamProcOrder.CloseSend()

	// 等待协程退出
	<-channel
}

func asncClientBidirectionalRPC(streamProcOder pb.OrderManagement_ProcessOrdersClient, c chan struct{}) {
	for {
		combinedShipment, errProcOrder := streamProcOder.Recv()
		if errProcOrder != nil {
			log.Printf("Error Receiving messages %v", errProcOrder)
			break
		} else {
			if errProcOrder == io.EOF {
				log.Printf("EOF")
				break
			}
			log.Printf("Combined shipment : %s", combinedShipment.OrdersList)
		}
	}
	c <- struct{}{}
}

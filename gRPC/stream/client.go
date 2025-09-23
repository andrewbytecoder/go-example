package main

import (
	"context"
	"io"
	"log"
	"time"

	pb "github.com/go-example/gRPC/stream/pb"
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
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(orderUnaryClientInterceptor),
		grpc.WithStreamInterceptor(clientStreamInterceptor))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Add Order
	order1 := pb.Order{Id: "101", Items: []string{"iPhone XS", "Mac Book Pro"}, Destination: "San Jose, CA", Price: 2300.00}
	res, _ := client.AddOrder(ctx, &order1)
	if res != nil {
		log.Print("AddOrder Response -> ", res.Value)
	}

	// Get Order
	retrievedOrder, err := client.GetOrder(ctx, &wrapper.StringValue{Value: "106"})
	log.Print("GetOrder Response -> : ", retrievedOrder)

	// Search Order : Server streaming scenario
	// 如果返回值是一个stream类型的值，那么需要循环接收数据直到stream关闭
	searchStream, _ := client.SearchOrders(ctx, &wrapper.StringValue{Value: "Google"})
	for {
		searchOrder, err := searchStream.Recv()
		if err == io.EOF {
			log.Print("EOF")
			break
		}

		if err == nil {
			log.Print("Search Result : ", searchOrder)
		}
	}

	// =========================================
	// Update Orders : Client streaming scenario
	updOrder1 := pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}
	updOrder2 := pb.Order{Id: "103", Items: []string{"Apple Watch S4", "Mac Book Pro", "iPad Pro"}, Destination: "San Jose, CA", Price: 2800.00}
	updOrder3 := pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub", "iPad Mini"}, Destination: "Mountain View, CA", Price: 2200.00}
	//
	updateStream, err := client.UpdateOrders(ctx)
	if err != nil {
		log.Fatalf("%v.UpdateOrders(_) = _, %v", client, err)
	}

	// Updating order 1
	if err := updateStream.Send(&updOrder1); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder1, err)
	}
	//
	// Updating order 2
	if err := updateStream.Send(&updOrder2); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder2, err)
	}
	//
	// Updating order 3
	if err := updateStream.Send(&updOrder3); err != nil {
		log.Fatalf("%v.Send(%v) = %v", updateStream, updOrder3, err)
	}
	// 必须调用，也只能调用一次
	updateRes, err := updateStream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", updateStream, err, nil)
	}
	log.Printf("Update Orders Res : %s", updateRes)

	// =========================================
	// Process Order : Bi-di streaming scenario
	streamProcOrder, err := client.ProcessOrders(ctx)
	if err != nil {
		log.Fatalf("%v.ProcessOrders(_) = _, %v", client, err)
	}

	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "102"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "102", err)
	}

	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "103"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "103", err)
	}

	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "104"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "104", err)
	}

	channel := make(chan struct{})
	go asncClientBidirectionalRPC(streamProcOrder, channel)
	time.Sleep(time.Millisecond * 1000)
	//
	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "101"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "101", err)
	}
	// 客户端这边只能控制发送端的关闭，之后服务端能控制返回数据的关闭
	if err := streamProcOrder.CloseSend(); err != nil {
		log.Fatal(err)
	}
	channel <- struct{}{}
}

func asncClientBidirectionalRPC(streamProcOrder pb.OrderManagement_ProcessOrdersClient, c chan struct{}) {
	for {
		combinedShipment, errProcOrder := streamProcOrder.Recv()
		if errProcOrder == io.EOF {
			break
		}
		log.Printf("Combined shipment : ", combinedShipment.OrdersList)
	}
	// 使用chan控制协程的退出
	<-c
}

func orderUnaryClientInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	log.Println("[Order Unary Client Interceptor]")
	return invoker(ctx, method, req, reply, cc, opts...)
}

type wrappedStreamClient struct {
	grpc.ClientStream
}

func (w *wrappedStreamClient) RecvMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor] Receive a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.RecvMsg(m)
}

func (w *wrappedStreamClient) SendMsg(m interface{}) error {
	log.Printf("====== [Client Stream Interceptor] Send a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.SendMsg(m)
}

func newWrappedStreamClient(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStreamClient{s}
}

func clientStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

	log.Println("======= [Client Interceptor] ", method)
	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}
	return newWrappedStreamClient(s), nil
}

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"io"
	"log"
	"net"
	"strings"

	/*"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"*/
	pb "github.com/go-example/gRPC/stream/pb"

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
	return &wrapper.StringValue{Value: "Order Added: " + orderReq.Id}, nil
}

// GetOrder Simple RPC
func (s *server) GetOrder(ctx context.Context, orderId *wrapper.StringValue) (*pb.Order, error) {
	ord, exists := orderMap[orderId.Value]
	if exists {
		return &ord, status.New(codes.OK, "").Err()
	}

	return nil, status.Errorf(codes.NotFound, "Order does not exist. : ", orderId)

}

// SearchOrders Server-side Streaming RPC
func (s *server) SearchOrders(searchQuery *wrappers.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {

	for key, order := range orderMap {
		log.Print(key, order)
		for _, itemStr := range order.Items {
			log.Print(itemStr)
			if strings.Contains(itemStr, searchQuery.Value) {
				// Send the matching orders in a stream
				err := stream.Send(&order)
				if err != nil {
					return fmt.Errorf("error sending message to stream : %v", err)
				}
				log.Print("Matching Order Found : " + key)
				break
			}
		}
	}
	return nil
}

// UpdateOrders Client-side Streaming RPC
func (s *server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {

	ordersStr := "Updated Order IDs : "
	for {
		order, err := stream.Recv()
		if err == io.EOF {
			// Finished reading the order stream.
			return stream.SendAndClose(&wrapper.StringValue{Value: "Orders processed " + ordersStr})
		}

		if err != nil {
			return err
		}
		// Update order
		orderMap[order.Id] = *order

		log.Printf("Order ID : %s - %s", order.Id, "Updated")
		ordersStr += order.Id + ", "
	}
}

// ProcessOrders Bi-directional Streaming RPC
func (s *server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {

	batchMarker := 1
	var combinedShipmentMap = make(map[string]pb.CombinedShipment)
	for {
		orderId, err := stream.Recv()
		log.Printf("Reading Proc order : %s", orderId)
		if err == io.EOF {
			// Client has sent all the messages
			// Send remaining shipments
			log.Printf("EOF : %s", orderId)
			for _, shipment := range combinedShipmentMap {
				if err := stream.Send(&shipment); err != nil {
					return err
				}
			}
			return nil
		}
		if err != nil {
			log.Println(err)
			return err
		}

		destination := orderMap[orderId.GetValue()].Destination
		shipment, found := combinedShipmentMap[destination]

		if found {
			ord := orderMap[orderId.GetValue()]
			shipment.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = shipment
		} else {
			comShip := pb.CombinedShipment{Id: "cmb - " + (orderMap[orderId.GetValue()].Destination), Status: "Processed!"}
			ord := orderMap[orderId.GetValue()]
			comShip.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = comShip
			log.Print(len(comShip.OrdersList), comShip.GetId())
		}

		if batchMarker == orderBatchSize {
			for _, comb := range combinedShipmentMap {
				log.Printf("Shipping : %v -> %v", comb.Id, len(comb.OrdersList))
				if err := stream.Send(&comb); err != nil {
					return err
				}
			}
			batchMarker = 0
			combinedShipmentMap = make(map[string]pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}

func orderServerUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	log.Println("OrderServerUnaryInterceptor")
	log.Println("======= [Server Interceptor] ", info.FullMethod)
	log.Printf(" Pre Proc Message : %s", req)
	// Invoking the handler to complete the normal execution of a unary RPC.
	// 触发 RPC调用， 当添加拦截器后，拦截器会先执行，然后才会执行 RPC调用
	// 添加拦截器之后， 需要手动调用 RPC才会执行RPC调用
	m, err := handler(ctx, req)

	//Post-processing logic
	log.Printf(" Post Proc Message : %s", m)
	return m, err
	//return nil, nil
}

// wrappedStream wraps around the embedded grpc.ServerStream, and intercepts the RecvMsg and
// SendMsg method call.
type wrappedStream struct {
	grpc.ServerStream
}

// 使用继承实现发送和接收函数的消息拦截

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] Receive a message (Type: %T) at %s", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf("====== [Server Stream Interceptor Wrapper] Send a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ServerStream.SendMsg(m)
}

func newWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s}
}

func orderServerStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("OrderServerStreamInterceptor")
	log.Println("======= [Server Interceptor] ", info.FullMethod)

	nSs := newWrappedStream(ss)

	err := handler(srv, nSs)
	log.Println("======= [Server Interceptor] ", info.FullMethod, "error: ", err)
	// 添加拦截器之后， 需要手动调用 RPC才会执行RPC调用
	return err
}

func main() {
	initSampleData()
	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.UnaryInterceptor(orderServerUnaryInterceptor),
		grpc.StreamInterceptor(orderServerStreamInterceptor))
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

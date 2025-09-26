package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
	/*"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"*/
	pb "github.com/go-example/gRPC/grpcprometheus/pb"

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
	// 每次访问对应的value labels 增加一次
	customizedCounterMetric.WithLabelValues(orderReq.Id).Inc()

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

// SearchOrders Server-side Streaming RPC
func (s *server) SearchOrders(searchQuery *wrapper.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {

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
		// 阻塞住，等待客户端发来数据，或者cancel掉
		orderId, err := stream.Recv()
		// 查看当前的RCP是否被对端取消
		if errors.Is(stream.Context().Err(), context.Canceled) {
			// 如果客户端调用了取消，这里直接退出不再读取数据
			log.Printf("Context Cacelled for this stream: -> %s", stream.Context().Err())
			log.Printf("Stopped processing any more order of this stream!")
			return stream.Context().Err()
		}
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

var (
	// create a metrics registry
	reg = prometheus.NewRegistry()

	// create some standard server metrics.
	grpcMetrics = grpc_prometheus.NewServerMetrics()

	customizedCounterMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "product_mgt_server_handle_count",
		Help: "Total number of RPCs handled on the server.",
	}, []string{"method"})
)

func init() {
	// Register standard server metrics and customized metrics to registry
	reg.MustRegister(grpcMetrics, customizedCounterMetric)
}

func main() {
	initSampleData()
	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Crate a http server for prometheus metrics
	httpServer := &http.Server{Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}), Addr: fmt.Sprintf("0.0.0.0:%d", 9092)}

	s := grpc.NewServer(
		grpc.StreamInterceptor(grpcMetrics.StreamServerInterceptor()),
		grpc.UnaryInterceptor(grpcMetrics.UnaryServerInterceptor()))
	pb.RegisterOrderManagementServer(s, &server{})

	// initialize all metrics
	grpcMetrics.InitializeMetrics(s)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start prometheus http server: %v", err)
		}
	}()

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

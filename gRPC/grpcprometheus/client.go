package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	pb "github.com/go-example/gRPC/grpcprometheus/pb"
	wrapper "github.com/golang/protobuf/ptypes/wrappers"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

const (
	address = "127.0.0.1:50051"
)

func main() {
	// create a metrics registry
	reg := prometheus.NewRegistry()
	// create some standard metrics
	grpcMetrics := grpc_prometheus.NewClientMetrics()
	// Register client metrics to registry
	reg.MustRegister(grpcMetrics)

	// Setting up a connection to the server.
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// Crate a http server for prometheus
	httpsServer := &http.Server{Handler: promhttp.HandlerFor(reg, promhttp.HandlerOpts{}), Addr: fmt.Sprintf("0.0.0.0:%d", 9094)}
	// start your http server for prometheus
	go func() {
		if err := httpsServer.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start prometheus http server: %v", err)
		}
	}()

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

package main

import (
	"log"
	"net"

	pb "github.com/go-example/gRPC/sendmessage/pb"

	"golang.org/x/net/context"
	// 导入grpc包
	"google.golang.org/grpc"
	// 导入刚才我们生成的代码所在的proto包。
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedSendMessageServer
}

// UnimplementedGreeterServer must be embedded to have
// forwarded compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGreeterServer struct{}

func (server) AddProduct(ctx context.Context, in *pb.Product) (*pb.ProductID, error) {
	log.Printf("Received: %v", in.GetName())
	log.Println("AddProduct", in)

	return &pb.ProductID{Id: in.Id}, nil
}

func main() {
	// 监听127.0.0.1:50051地址
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 实例化grpc服务端
	s := grpc.NewServer()

	// 注册Greeter服务
	pb.RegisterSendMessageServer(s, &server{})

	// 往grpc服务端注册反射服务
	reflection.Register(s)

	// 启动grpc服务
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

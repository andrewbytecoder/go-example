package main

import (
	"log"
	"net"

	pb "github.com/go-example/gRPC/sendmessage/pb"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// 导入grpc包
	"google.golang.org/grpc"
	// 导入刚才我们生成的代码所在的proto包。
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedSendMessageServer
	products map[int32]*pb.Product
}

// UnimplementedGreeterServer must be embedded to have
// forwarded compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGreeterServer struct{}

func (s *server) AddProduct(ctx context.Context, in *pb.Product) (*pb.ProductID, error) {
	log.Printf("Received: %v", in.GetName())
	log.Println("AddProduct", in)

	s.products[in.Id] = in

	return &pb.ProductID{Id: in.Id}, nil
}

func (s *server) GetProduct(ctx context.Context, in *pb.ProductID) (*pb.Product, error) {
	product, ok := s.products[in.Id]
	if !ok {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	// 如果查找到产品Id就直接返回产品ID如果查询直接返回错误原因
	return &pb.Product{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
	}, nil
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
	pb.RegisterSendMessageServer(s, &server{
		products: make(map[int32]*pb.Product),
	})

	// 往grpc服务端注册反射服务
	reflection.Register(s)

	// 启动grpc服务
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

package main

import (
	"log"
	"net"

	pb "github.com/go-example/gRPC/sendmessage/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/net/context"
	// 导入grpc包
	"google.golang.org/grpc"
	// 导入刚才我们生成的代码所在的proto包。
	"google.golang.org/grpc/reflection"
)

// # 切换到helloworld项目根目录，执行命令
// protoc -I proto/ --go_out=plugins=grpc:proto proto/helloworld.proto
// -I 指定代码输出目录，忽略服务定义的包名，否则会根据包名创建目录
// --go_out 指定代码输出目录，格式：--go_out=plugins=grpc:目录名
// 命令最后面的参数是proto协议文件

// # 生成go代码
//  protoc -I proto/ --go_out=proto --go-grpc_out=proto proto/helloworld.proto

// 定义server，用来实现proto文件，里面实现的Greeter服务里面的接口
type server struct {
	pb.UnimplementedSendMessageServer
}

// UnimplementedGreeterServer must be embedded to have
// forwarded compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGreeterServer struct{}

func (server) SendMessage(ctx context.Context, in *pb.Request) (*pb.Response, error) {
	log.Printf("Received: %v", in.GetName())
	return nil, status.Errorf(codes.Unimplemented, "method SayHello not implemented")
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

package main

import (
	"flag"
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc/credentials/insecure"

	// 导入grpc包
	"google.golang.org/grpc"
	// 导入刚才我们生成的代码所在的proto包。
	pb "github.com/go-example/gRPC/sendmessage/pb"
)

const (
	defaultName = "world"
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.NewClient(":8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewSendMessageClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.AddProduct(ctx, &pb.Product{Id: 1, Name: "product id is 1", Description: "add production"})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Add product id: %d", r.Id)

	p, err := c.GetProduct(ctx, &pb.ProductID{Id: 1})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Get product: %s", p.Name)
}

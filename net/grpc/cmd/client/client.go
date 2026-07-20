package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	pb "github.com/go-example/net/grpc/tunnel"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// TunnelClient 机房2的代理客户端，接收机房1发来的TunnelRequest并转发到本地服务.
type TunnelClient struct {
	stream   pb.TunnelService_OpenTunnelClient
	respChan map[string]chan *pb.TunnelResponse
	mu       sync.Mutex
}

// recvLoop 持续接收来自机房1的TunnelRequest，发起本地HTTP调用后返回TunnelResponse.
func (c *TunnelClient) recvLoop() {
	for {
		req, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("recvLoop: stream closed by server")
			return
		}
		if err != nil {
			log.Printf("recvLoop error: %v", err)
			return
		}

		go c.handleRequest(req)
	}
}

func (c *TunnelClient) handleRequest(req *pb.TunnelRequest) {
	httpReq, err := http.NewRequest(req.Method, req.Url, strings.NewReader(string(req.Body)))
	if err != nil {
		c.sendError(req.Id, http.StatusBadRequest, err.Error())
		return
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.sendError(req.Id, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	c.stream.Send(&pb.TunnelResponse{
		Id:         req.Id,
		StatusCode: int32(resp.StatusCode),
		Headers:    respHeaders,
		Body:       body,
	})
}

func (c *TunnelClient) sendError(id string, statusCode int32, msg string) {
	c.stream.Send(&pb.TunnelResponse{
		Id:         id,
		StatusCode: statusCode,
		Body:       []byte(msg),
	})
}

// ServeHTTP 提供简单的健康检查接口.
func (c *TunnelClient) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("agent running"))
}

func StartAgent(prefix string, grpcTarget string) {
	// 1. 建立 gRPC 连接.
	conn, err := grpc.NewClient(grpcTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}

	// 2. 在 Context 中注入 Prefix 元数据.
	md := metadata.Pairs("prefix", prefix)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// 3. 建立双向流.
	client := pb.NewTunnelServiceClient(conn)
	stream, err := client.OpenTunnel(ctx)
	if err != nil {
		log.Fatalf("OpenTunnel failed: %v", err)
	}

	log.Printf("代理已启动，注册前缀: %s", prefix)

	// 4. 启动本地 HTTP 服务，绑定到该流.
	agent := &TunnelClient{stream: stream, respChan: make(map[string]chan *pb.TunnelResponse)}
	go agent.recvLoop() // 后台接收请求

	log.Fatal(http.ListenAndServe(":0", agent))
}

func main() {
	// 启动多个代理实例
	go StartAgent("/app-a", "机房1_IP:50051")
	go StartAgent("/app-b", "机房1_IP:50051")
	go StartAgent("/file-service", "机房1_IP:50051")

	select {} // 阻塞主协程
}

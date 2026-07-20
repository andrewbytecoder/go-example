这是一个非常典型的多租户/多路由网关架构。在这种场景下，我们需要将“路由注册”与“数据透传”分离开来。

核心设计思路如下：
注册机制：机房2的每个代理在启动时，通过一个独立的 RPC（如 Register）向机房1上报自己的 URL前缀。
路由表：机房1维护一个并发安全的映射表（map[string]Stream），将 URL 前缀与对应的 gRPC 双向流绑定。
数据下发：机房1收到本地应用的请求时，解析 URL 前缀，从路由表中找到对应的流，将请求 Send 过去。
生命周期管理：当机房2断开连接或流异常时，机房1必须从路由表中清除该前缀，防止内存泄漏和路由错乱。

以下是基于 Go 语言的完整实现方案：

Proto 定义 (tunnel.proto)
增加一个专门用于注册的 RPC 方法。

syntax = "proto3";

package tunnel;
option go_package = "./pb";

// 注册请求
message RegisterRequest {
string prefix = 1;      // URL前缀，例如 "/app-a"
string agent_id = 2;    // 代理唯一标识（可选，用于日志追踪）
}

message RegisterResponse {
bool success = 1;
string message = 2;
}

// 隧道透传消息
message TunnelRequest {
string id = 1;
string method = 2;
string url = 3;
map<string, string> headers = 4;
bytes body = 5;
}

message TunnelResponse {
string id = 1;
int32 status_code = 2;
map<string, string> headers = 3;
bytes body = 4;
}

service TunnelService {
// 1. 注册路由前缀
rpc Register(RegisterRequest) returns (RegisterResponse);

// 2. 建立数据透传双向流
rpc OpenTunnel(stream TunnelRequest) returns (stream TunnelResponse);
}

机房1：核心路由服务端 (Server)
机房1需要维护路由表，并处理并发下发。

package main

import (
"context"
"fmt"
"io"
"log"
"net/http"
"strings"
"sync"
"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	pb "your_project/pb"
)

type TunnelServer struct {
pb.UnimplementedTunnelServiceServer

	mu      sync.RWMutex
	routes  map[string]pb.TunnelService_OpenTunnelServer // 前缀 -> 流句柄
}

func NewTunnelServer() *TunnelServer {
return &TunnelServer{
routes: make(map[string]pb.TunnelService_OpenTunnelServer),
}
}

// 1. 处理注册逻辑
func (s *TunnelServer) Register(ctx context.Context, reqpb.RegisterRequest) (pb.RegisterResponse, error) {
s.mu.Lock()
defer s.mu.Unlock()

	if _, exists := s.routes[req.Prefix]; exists {
		return &pb.RegisterResponse{Success: false, Message: "Prefix already registered"}, nil
	}
	
	// 注意：此时还没有流，注册只是记录意图，真正的绑定在 OpenTunnel 中完成
	// 或者，我们可以要求客户端在 Register 成功后立刻调用 OpenTunnel
	log.Printf("收到注册请求: Prefix=%s, Agent=%s", req.Prefix, req.AgentId)
	return &pb.RegisterResponse{Success: true, Message: "OK"}, nil
}

// 2. 处理双向流，绑定路由并持续接收响应
func (s *TunnelServer) OpenTunnel(stream pb.TunnelService_OpenTunnelServer) error {
// 从流的 Context 中提取元数据，获取客户端注册的 Prefix
// 实际生产中，建议在 Dial 时通过 Metadata 传递 Prefix，或者在第一个 TunnelRequest 中传递
md, ok := metadata.FromIncomingContext(stream.Context())
if !ok || len(md["prefix"]) == 0 {
return status.Error(codes.InvalidArgument, "Missing prefix in metadata")
}
prefix := md["prefix"][0]

	// 绑定路由
	s.mu.Lock()
	if _, exists := s.routes[prefix]; exists {
		s.mu.Unlock()
		return status.Error(codes.AlreadyExists, "Prefix already has an active stream")
	}
	s.routes[prefix] = stream
	s.mu.Unlock()
	log.Printf("路由绑定成功: Prefix=%s", prefix)

	// 确保流断开时，清理路由表
	defer func() {
		s.mu.Lock()
		delete(s.routes, prefix)
		s.mu.Unlock()
		log.Printf("路由已解绑: Prefix=%s", prefix)
	}()

	// 持续监听来自机房2的响应（这部分逻辑与之前相同）
	for {
		resp, err := stream.Recv()
		if err == io.EOF { return nil }
		if err != nil { return err }
		
		// 将响应分发回对应的本地 HTTP 请求（通过 resp.Id 匹配）
		dispatchResponse(resp) 
	}
}

// 3. 供本地应用调用的 HTTP 代理接口（核心路由分发）
func (s *TunnelServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
prefix := extractPrefix(r.URL.Path) // 自定义函数，提取路径前缀

	s.mu.RLock()
	stream, ok := s.routes[prefix]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("No agent registered for prefix: %s", prefix), http.StatusBadGateway)
		return
	}

	// 构造请求并发送
	id := uuid.New().String()
	req := &pb.TunnelRequest{
		Id: id, Method: r.Method, Url: r.URL.String(),
		Headers: make(map[string]string),
	}
	for k := range r.Header { req.Headers[k] = r.Header.Get(k) }
	body, _ := io.ReadAll(r.Body)
	req.Body = body

	// 注册等待通道
	ch := make(chan *pb.TunnelResponse, 1)
	registerWaiter(id, ch) // 注册到全局的响应等待池
	defer unregisterWaiter(id)

	if err := stream.Send(req); err != nil {
		http.Error(w, "Send to tunnel failed", http.StatusBadGateway)
		return
	}

	// 阻塞等待响应
	select {
	case resp := <-ch:
		for k, v := range resp.Headers { w.Header().Set(k, v) }
		w.WriteHeader(int(resp.StatusCode))
		w.Write(resp.Body)
	case <-time.After(35 * time.Second):
		http.Error(w, "Tunnel Timeout", http.StatusGatewayTimeout)
	}
}

func main() {
server := NewTunnelServer()

	kp := keepalive.ServerParameters{Time: 30 * time.Second, Timeout: 10 * time.Second}
	enforcement := keepalive.EnforcementPolicy{MinTime: 10 * time.Second, PermitWithoutStream: true}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kp),
		grpc.KeepaliveEnforcementPolicy(enforcement),
	)
	pb.RegisterTunnelServiceServer(grpcServer, server)

	// 同时启动 gRPC 和 HTTP 服务
	go func() { grpcServer.Serve(...) }()
	http.ListenAndServe(":80", server)
}

机房2：多实例代理客户端 (Client)
机房2的每个应用启动一个独立的代理实例，通过 Metadata 传递前缀。

package main

import (
"context"
"log"
"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	pb "your_project/pb"
)

func StartAgent(prefix string, grpcTarget string) {
// 1. 建立 gRPC 连接
conn, err := grpc.Dial(grpcTarget, grpc.WithInsecure())
if err != nil { log.Fatalf("Dial failed: %v", err) }

	// 2. 在 Context 中注入 Prefix 元数据
	md := metadata.Pairs("prefix", prefix)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	// 3. 建立双向流
	client := pb.NewTunnelServiceClient(conn)
	stream, err := client.OpenTunnel(ctx)
	if err != nil { log.Fatalf("OpenTunnel failed: %v", err) }
	
	log.Printf("代理已启动，注册前缀: %s", prefix)

	// 4. 启动本地 HTTP 服务，绑定到该流
	agent := &TunnelClient{stream: stream, respChan: make(map[string]chan *pb.TunnelResponse)}
	go agent.recvLoop() // 后台接收响应
	
	http.ListenAndServe(":0", agent) // :0 表示随机分配可用端口，或指定固定端口
}

func main() {
// 启动多个代理实例
go StartAgent("/app-a", "机房1_IP:50051")
go StartAgent("/app-b", "机房1_IP:50051")
go StartAgent("/file-service", "机房1_IP:50051")

	select {} // 阻塞主协程
}

架构关键点总结：
Metadata 传递前缀：不要在每个 TunnelRequest 里都带上前缀，这会增加带宽开销。利用 gRPC 的 metadata 在建立流（OpenTunnel）时传递一次即可。
读写锁 (sync.RWMutex)：机房1的路由表是高频读、低频写的场景。使用读写锁可以确保多个 HTTP 请求并发下发时不会互相阻塞。
自动清理机制：利用 defer 在 OpenTunnel 退出时自动清理路由表，这是防止内存泄漏和僵尸路由的最安全做法。
响应分发池：机房1需要一个全局的 map[string]chan *pb.TunnelResponse（即代码中的 registerWaiter），用于根据 resp.Id 将流中收到的响应精准投递给正在等待的 HTTP Handler。

这套架构具备极强的扩展性，机房2可以随时动态增加或减少应用代理，机房1的路由表会自动感知并更新。

机房2中每个应用启动一个代理实例，每个实例都会独立建立一条 gRPC 长连接，机房1那边连接数可能会比较多。需要我帮你补充连接池复用（一个代理实例复用一条 gRPC 连接）的配置示例吗？
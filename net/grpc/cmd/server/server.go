package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/go-example/net/grpc/tunnel"
)

type TunnelServer struct {
	pb.UnimplementedTunnelServiceServer

	mu     sync.RWMutex
	routes map[string]pb.TunnelService_OpenTunnelServer // 前缀 -> 流句柄
}

func NewTunnelServer() *TunnelServer {
	return &TunnelServer{
		routes: make(map[string]pb.TunnelService_OpenTunnelServer),
	}
}

// Register 处理注册逻辑.
func (s *TunnelServer) Register(_ context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
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

// OpenTunnel 处理双向流，绑定路由并持续接收响应.
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
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// 将响应分发回对应的本地 HTTP 请求（通过 resp.Id 匹配）
		dispatchResponse(resp)
	}
}

// 3. 供本地应用调用的 HTTP 代理接口（核心路由分发）
func (s *TunnelServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	prefix := extractPrefix(r.URL.Path, s.routes) // 自定义函数，提取路径前缀
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
	for k := range r.Header {
		req.Headers[k] = r.Header.Get(k)
	}
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
		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(int(resp.StatusCode))
		_, _ = w.Write(resp.Body)
	case <-time.After(35 * time.Second):
		http.Error(w, "Tunnel Timeout", http.StatusGatewayTimeout)
	}
}

var (
	waiterMu     sync.Mutex
	responsePool = make(map[string]chan *pb.TunnelResponse)
)

// registerWaiter 注册一个等待通道，供 HTTP Handler 阻塞等待响应
func registerWaiter(id string, ch chan *pb.TunnelResponse) {
	waiterMu.Lock()
	defer waiterMu.Unlock()
	responsePool[id] = ch
}

// unregisterWaiter 移除等待通道，防止内存泄漏
// 必须在 HTTP Handler 的 defer 中调用
func unregisterWaiter(id string) {
	waiterMu.Lock()
	defer waiterMu.Unlock()
	delete(responsePool, id)
}

// dispatchResponse 由 gRPC 流的 Recv 循环调用，将收到的响应精准分发
func dispatchResponse(resp *pb.TunnelResponse) {
	waiterMu.Lock()
	ch, ok := responsePool[resp.Id]
	waiterMu.Unlock()

	if ok {
		// 使用非阻塞发送，防止 Handler 已经超时退出导致 Recv 循环阻塞
		select {
		case ch <- resp:
		default:
		}
	}
}

//	func (s *TunnelServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//		// 1. 提取前缀（传入路由表进行最长匹配）
//		prefix := extractPrefix(r.URL.Path, s.routes)
//		if prefix == "" {
//			http.Error(w, "No matching route found", http.StatusNotFound)
//			return
//		}
//
//		// 2. 查找对应的流
//		s.mu.RLock()
//		stream, ok := s.routes[prefix]
//		s.mu.RUnlock()
//
//		if !ok {
//			http.Error(w, "Agent disconnected", http.StatusBadGateway)
//			return
//		}
//
//		// 3. 生成唯一 ID 并注册等待通道
//		id := uuid.New().String()
//		ch := make(chan *pb.TunnelResponse, 1)
//		registerWaiter(id, ch)
//		defer unregisterWaiter(id) // 【关键】确保无论成功还是超时，都清理等待池
//
//		// 4. 构造并发送请求... (省略，与之前一致)
//		if err := stream.Send(req); err != nil {
//			http.Error(w, "Send failed", http.StatusBadGateway)
//			return
//		}
//
//		// 5. 阻塞等待响应
//		select {
//		case resp := <-ch:
//			// 正常返回响应...
//		case <-time.After(35 * time.Second):
//			http.Error(w, "Timeout", http.StatusGatewayTimeout)
//		}
//	}
//
//	func extractPrefix(path string, routes map[string]pb.TunnelService_OpenTunnelServer) string {
//		// 按 "/" 分割路径，例如 /app-a/api/v1 -> ["", "app-a", "api", "v1"]
//		parts := strings.Split(strings.Trim(path, "/"), "/")
//		if len(parts) == 0 {
//			return ""
//		}
//
//		// 尝试用第一段匹配
//		candidate := "/" + parts[0]
//		if _, ok := routes[candidate]; ok {
//			return candidate
//		}
//		return ""
//	}
//
// extractPrefix 从请求路径中提取匹配的前缀
// routes 是机房1维护的路由表 map[string]Stream
func extractPrefix(path string, routes map[string]pb.TunnelService_OpenTunnelServer) string {
	var matchedPrefix string
	maxLen := 0

	// 遍历所有已注册的前缀
	for prefix := range routes {
		// 检查请求路径是否以该前缀开头
		if strings.HasPrefix(path, prefix) {
			// 保留最长的那个前缀
			if len(prefix) > maxLen {
				maxLen = len(prefix)
				matchedPrefix = prefix
			}
		}
	}
	return matchedPrefix // 如果没有匹配到，返回空字符串 ""
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
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()
	log.Println("gRPC server listening on :9090")
	log.Fatal(http.ListenAndServe(":80", server))
}

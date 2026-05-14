package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

// DiscoveryService 节点发现服务
type DiscoveryService struct {
	multicastAddr string
	port          int
	nodeInfo      *NodeInfo
	handler       func(*NodeInfo) error
}

// NodeInfo 节点信息
type NodeInfo struct {
	Name       string            `json:"name"`
	Address    string            `json:"address"`
	APIPort    int               `json:"apiPort"`
	Role       string            `json:"role"` // server 或 agent
	CPU        string            `json:"cpu"`
	Memory     string            `json:"memory"`
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Labels     map[string]string `json:"labels"`
	Timestamp  int64             `json:"timestamp"`
}

// NewDiscoveryService 创建发现服务
func NewDiscoveryService(nodeName, address string, apiPort int, role string) *DiscoveryService {
	cpu := "4"
	memory := "8192Mi"
	
	return &DiscoveryService{
		multicastAddr: "239.255.0.1",
		port:          7946,
		nodeInfo: &NodeInfo{
			Name:    nodeName,
			Address: address,
			APIPort: apiPort,
			Role:    role,
			CPU:     cpu,
			Memory:  memory,
			OS:      runtime.GOOS,
			Arch:    runtime.GOARCH,
			Labels: map[string]string{
				"kubernetes.io/os":   runtime.GOOS,
				"kubernetes.io/arch": runtime.GOARCH,
			},
			Timestamp: time.Now().Unix(),
		},
	}
}

// SetHandler 设置节点发现回调
func (d *DiscoveryService) SetHandler(handler func(*NodeInfo) error) {
	d.handler = handler
}

// Start 启动发现服务
func (d *DiscoveryService) Start(ctx context.Context) error {
	// 启动广播
	go d.broadcastLoop(ctx)
	
	// 启动监听
	return d.listenLoop(ctx)
}

// broadcastLoop 定期广播节点信息
func (d *DiscoveryService) broadcastLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	// 立即广播一次
	d.broadcast()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.broadcast()
		}
	}
}

// broadcast 广播节点信息
func (d *DiscoveryService) broadcast() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", d.multicastAddr, d.port))
	if err != nil {
		return
	}
	
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()
	
	// 更新时间戳
	d.nodeInfo.Timestamp = time.Now().Unix()
	
	// 序列化节点信息
	data := fmt.Sprintf("MINIK8S|%s|%s|%d|%s|%s|%s|%s|%s|%d",
		d.nodeInfo.Name,
		d.nodeInfo.Address,
		d.nodeInfo.APIPort,
		d.nodeInfo.Role,
		d.nodeInfo.CPU,
		d.nodeInfo.Memory,
		d.nodeInfo.OS,
		d.nodeInfo.Arch,
		d.nodeInfo.Timestamp,
	)
	
	conn.Write([]byte(data))
}

// listenLoop 监听多播消息
func (d *DiscoveryService) listenLoop(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", d.multicastAddr, d.port))
	if err != nil {
		return err
	}
	
	// 绑定到所有接口
	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	
	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	
	buffer := make([]byte, 1024)
	
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		
		// 重置超时
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		
		n, src, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			continue
		}
		
		// 忽略自己的消息
		if src.IP.String() == d.nodeInfo.Address {
			continue
		}
		
		// 解析消息
		msg := string(buffer[:n])
		nodeInfo := d.parseMessage(msg)
		if nodeInfo != nil && d.handler != nil {
			// 异步处理
			go func() {
				if err := d.handler(nodeInfo); err != nil {
					log.Printf("Discovery handler error: %v", err)
				}
			}()
		}
	}
}

// parseMessage 解析广播消息
func (d *DiscoveryService) parseMessage(msg string) *NodeInfo {
	// 格式: MINIK8S|name|address|apiPort|role|cpu|memory|os|arch|timestamp
	parts := splitMessage(msg, '|')
	if len(parts) != 10 || parts[0] != "MINIK8S" {
		return nil
	}
	
	var apiPort int
	fmt.Sscanf(parts[3], "%d", &apiPort)
	
	var timestamp int64
	fmt.Sscanf(parts[9], "%d", &timestamp)
	
	return &NodeInfo{
		Name:      parts[1],
		Address:   parts[2],
		APIPort:   apiPort,
		Role:      parts[4],
		CPU:       parts[5],
		Memory:    parts[6],
		OS:        parts[7],
		Arch:      parts[8],
		Timestamp: timestamp,
	}
}

// splitMessage 分割消息
func splitMessage(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// GetLocalIP 获取本地 IP 地址
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	
	return "127.0.0.1"
}

// GetHostname 获取主机名
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// DiscoveryManager 发现管理器（Server 端使用）
type DiscoveryManager struct {
	nodes map[string]*NodeInfo
	mu    sync.RWMutex
}

// NewDiscoveryManager 创建发现管理器
func NewDiscoveryManager() *DiscoveryManager {
	return &DiscoveryManager{
		nodes: make(map[string]*NodeInfo),
	}
}

// AddNode 添加节点
func (dm *DiscoveryManager) AddNode(node *NodeInfo) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	// 检查是否是新节点或更新
	existing, exists := dm.nodes[node.Name]
	if !exists || existing.Timestamp < node.Timestamp {
		dm.nodes[node.Name] = node
		log.Printf("Discovery: Node %s (%s) %s", node.Name, node.Address, map[bool]string{true: "updated", false: "discovered"}[exists])
	}
}

// GetNodes 获取所有发现的节点
func (dm *DiscoveryManager) GetNodes() []*NodeInfo {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	nodes := make([]*NodeInfo, 0, len(dm.nodes))
	for _, node := range dm.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// RemoveStaleNodes 移除过期节点
func (dm *DiscoveryManager) RemoveStaleNodes(timeout time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	now := time.Now().Unix()
	for name, node := range dm.nodes {
		if now-node.Timestamp > int64(timeout.Seconds()) {
			delete(dm.nodes, name)
			log.Printf("Discovery: Node %s removed (stale)", name)
		}
	}
}

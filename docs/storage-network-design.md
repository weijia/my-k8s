# MiniK8s 存储与网络设计文档

## 1. 存储设计

### 1.1 存储架构概述

MiniK8s 采用分层存储架构，支持多种存储类型以满足不同场景需求：

```
┌─────────────────────────────────────────────────────────────┐
│                      Storage Layer                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  emptyDir   │  │  hostPath   │  │  ConfigMap/Secret   │  │
│  │  (临时存储)  │  │  (主机挂载)  │  │    (配置存储)        │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │             │
│         └────────────────┼─────────────────────┘             │
│                          │                                   │
│                   ┌──────┴──────┐                           │
│                   │ Volume Mgr  │                           │
│                   │  (管理器)    │                           │
│                   └──────┬──────┘                           │
│                          │                                   │
│         ┌────────────────┼────────────────┐                  │
│         ▼                ▼                ▼                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Overlay   │  │    Bind     │  │   tmpfs     │          │
│  │   (联合文件) │  │   (挂载)     │  │  (内存文件)  │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 支持的存储类型

| 存储类型 | 生命周期 | 适用场景 | 跨节点 |
|----------|----------|----------|--------|
| **emptyDir** | Pod | 临时数据、缓存 | 否 |
| **hostPath** | 主机 | 主机文件访问、日志 | 否 |
| **ConfigMap** | 资源 | 配置文件 | 是 |
| **Secret** | 资源 | 敏感数据 | 是 |

### 1.3 emptyDir 实现

```go
// EmptyDirVolume 临时目录卷
type EmptyDirVolume struct {
    podUID    string
    volumeName string
    basePath  string
    medium    StorageMedium
}

// StorageMedium 存储介质
type StorageMedium string

const (
    StorageMediumDefault StorageMedium = ""       // 磁盘
    StorageMediumMemory  StorageMedium = "Memory" // tmpfs
)

// Setup 设置 emptyDir 卷
func (e *EmptyDirVolume) Setup() error {
    path := e.GetPath()
    
    if e.medium == StorageMediumMemory {
        // 使用 tmpfs
        return mountTmpfs(path, defaultTmpfsSize)
    }
    
    // 普通目录
    return os.MkdirAll(path, 0755)
}

// GetPath 获取卷路径
func (e *EmptyDirVolume) GetPath() string {
    return filepath.Join(e.basePath, "pods", e.podUID, "volumes", "kubernetes.io~empty-dir", e.volumeName)
}

// Teardown 清理 emptyDir 卷
func (e *EmptyDirVolume) Teardown() error {
    path := e.GetPath()
    
    if e.medium == StorageMediumMemory {
        // 卸载 tmpfs
        if err := unmount(path); err != nil {
            return err
        }
    }
    
    return os.RemoveAll(path)
}
```

### 1.4 hostPath 实现

```go
// HostPathVolume 主机路径卷
type HostPathVolume struct {
    podUID     string
    volumeName string
    path       string
    hostPath   string
    readOnly   bool
}

// Setup 设置 hostPath 卷
func (h *HostPathVolume) Setup() error {
    // 检查主机路径是否存在
    if _, err := os.Stat(h.hostPath); os.IsNotExist(err) {
        // 可选：自动创建目录
        if err := os.MkdirAll(h.hostPath, 0755); err != nil {
            return fmt.Errorf("failed to create host path: %v", err)
        }
    }
    
    // 创建挂载点
    mountPath := h.GetPath()
    if err := os.MkdirAll(mountPath, 0755); err != nil {
        return err
    }
    
    // 执行 bind mount
    flags := syscall.MS_BIND
    if h.readOnly {
        flags |= syscall.MS_RDONLY
    }
    
    return syscall.Mount(h.hostPath, mountPath, "", uintptr(flags), "")
}

// GetPath 获取卷路径
func (h *HostPathVolume) GetPath() string {
    return filepath.Join("/var/lib/minik8s/pods", h.podUID, "volumes", "kubernetes.io~host-path", h.volumeName)
}

// Teardown 清理 hostPath 卷
func (h *HostPathVolume) Teardown() error {
    mountPath := h.GetPath()
    
    // 卸载
    if err := syscall.Unmount(mountPath, 0); err != nil {
        return err
    }
    
    return os.RemoveAll(mountPath)
}
```

### 1.5 ConfigMap/Secret 实现

```go
// ConfigMapVolume ConfigMap 卷
type ConfigMapVolume struct {
    podUID     string
    volumeName string
    configMap  *ConfigMap
    basePath   string
}

// Setup 设置 ConfigMap 卷
func (c *ConfigMapVolume) Setup() error {
    path := c.GetPath()
    
    // 创建目录
    if err := os.MkdirAll(path, 0755); err != nil {
        return err
    }
    
    // 将每个数据项写入文件
    for key, value := range c.configMap.Data {
        filePath := filepath.Join(path, key)
        
        // 创建子目录（如果 key 包含路径）
        dir := filepath.Dir(filePath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
        
        if err := os.WriteFile(filePath, []byte(value), 0644); err != nil {
            return err
        }
    }
    
    return nil
}

// GetPath 获取卷路径
func (c *ConfigMapVolume) GetPath() string {
    return filepath.Join(c.basePath, "pods", c.podUID, "volumes", "kubernetes.io~configmap", c.volumeName)
}

// SecretVolume Secret 卷（类似 ConfigMap，但处理二进制数据）
type SecretVolume struct {
    podUID     string
    volumeName string
    secret     *Secret
    basePath   string
}

// Setup 设置 Secret 卷
func (s *SecretVolume) Setup() error {
    path := s.GetPath()
    
    if err := os.MkdirAll(path, 0755); err != nil {
        return err
    }
    
    // 将每个数据项写入文件（Secret 数据是 base64 编码的）
    for key, value := range s.secret.Data {
        filePath := filepath.Join(path, key)
        
        dir := filepath.Dir(filePath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
        
        // Secret 数据已经是 []byte
        if err := os.WriteFile(filePath, value, 0644); err != nil {
            return err
        }
    }
    
    return nil
}
```

### 1.6 Volume Manager

```go
// VolumeManager 卷管理器
type VolumeManager struct {
    volumes map[string]Volume
    mu      sync.RWMutex
    basePath string
}

// Volume 卷接口
type Volume interface {
    Setup() error
    GetPath() string
    Teardown() error
}

// NewVolumeManager 创建卷管理器
func NewVolumeManager(basePath string) *VolumeManager {
    return &VolumeManager{
        volumes:  make(map[string]Volume),
        basePath: basePath,
    }
}

// SetupPodVolumes 设置 Pod 的所有卷
func (vm *VolumeManager) SetupPodVolumes(pod *Pod) error {
    for _, vol := range pod.Spec.Volumes {
        volume, err := vm.createVolume(pod.UID, &vol)
        if err != nil {
            return fmt.Errorf("failed to create volume %s: %v", vol.Name, err)
        }
        
        if err := volume.Setup(); err != nil {
            return fmt.Errorf("failed to setup volume %s: %v", vol.Name, err)
        }
        
        key := fmt.Sprintf("%s/%s", pod.UID, vol.Name)
        vm.mu.Lock()
        vm.volumes[key] = volume
        vm.mu.Unlock()
    }
    
    return nil
}

// TeardownPodVolumes 清理 Pod 的所有卷
func (vm *VolumeManager) TeardownPodVolumes(pod *Pod) error {
    var errs []error
    
    for _, vol := range pod.Spec.Volumes {
        key := fmt.Sprintf("%s/%s", pod.UID, vol.Name)
        
        vm.mu.Lock()
        volume, exists := vm.volumes[key]
        if exists {
            delete(vm.volumes, key)
        }
        vm.mu.Unlock()
        
        if exists {
            if err := volume.Teardown(); err != nil {
                errs = append(errs, fmt.Errorf("failed to teardown volume %s: %v", vol.Name, err))
            }
        }
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("volume teardown errors: %v", errs)
    }
    
    return nil
}

// GetVolumePath 获取卷路径
func (vm *VolumeManager) GetVolumePath(podUID, volumeName string) (string, error) {
    key := fmt.Sprintf("%s/%s", podUID, volumeName)
    
    vm.mu.RLock()
    volume, exists := vm.volumes[key]
    vm.mu.RUnlock()
    
    if !exists {
        return "", fmt.Errorf("volume %s not found", volumeName)
    }
    
    return volume.GetPath(), nil
}

// createVolume 根据卷定义创建卷实例
func (vm *VolumeManager) createVolume(podUID string, vol *Volume) (Volume, error) {
    if vol.EmptyDir != nil {
        return &EmptyDirVolume{
            podUID:     podUID,
            volumeName: vol.Name,
            basePath:   vm.basePath,
            medium:     StorageMedium(vol.EmptyDir.Medium),
        }, nil
    }
    
    if vol.HostPath != nil {
        return &HostPathVolume{
            podUID:     podUID,
            volumeName: vol.Name,
            hostPath:   vol.HostPath.Path,
            readOnly:   false, // 可根据需要设置
        }, nil
    }
    
    if vol.ConfigMap != nil {
        // 需要获取 ConfigMap 对象
        configMap, err := vm.getConfigMap(vol.ConfigMap.Name)
        if err != nil {
            return nil, err
        }
        return &ConfigMapVolume{
            podUID:     podUID,
            volumeName: vol.Name,
            configMap:  configMap,
            basePath:   vm.basePath,
        }, nil
    }
    
    if vol.Secret != nil {
        secret, err := vm.getSecret(vol.Secret.SecretName)
        if err != nil {
            return nil, err
        }
        return &SecretVolume{
            podUID:     podUID,
            volumeName: vol.Name,
            secret:     secret,
            basePath:   vm.basePath,
        }, nil
    }
    
    return nil, fmt.Errorf("unsupported volume type")
}
```

---

## 2. 网络设计

### 2.1 网络架构概述

MiniK8s 采用简化但功能完整的网络架构：

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            Network Architecture                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                        Cluster Network                           │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │   │
│  │  │   Pod 1     │    │   Pod 2     │    │   Pod 3     │         │   │
│  │  │ 10.244.1.2  │    │ 10.244.1.3  │    │ 10.244.2.2  │         │   │
│  │  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘         │   │
│  │         │                  │                  │                 │   │
│  │         └──────────────────┼──────────────────┘                 │   │
│  │                            │                                    │   │
│  │                    ┌───────┴───────┐                            │   │
│  │                    │  CNI Bridge   │                            │   │
│  │                    │  (veth pair)  │                            │   │
│  │                    └───────┬───────┘                            │   │
│  │                            │                                    │   │
│  │                    ┌───────┴───────┐                            │   │
│  │                    │   cni0 bridge │                            │   │
│  │                    │  (10.244.1.1) │                            │   │
│  │                    └───────┬───────┘                            │   │
│  │                            │                                    │   │
│  │  ┌─────────────────────────┼─────────────────────────┐         │   │
│  │  │                         │                         │         │   │
│  │  ▼                         ▼                         ▼         │   │
│  │ ┌──────────┐          ┌──────────┐          ┌──────────┐      │   │
│  │ │  Node 1  │◄────────►│  Node 2  │◄────────►│  Node 3  │      │   │
│  │ │10.244.1.0│   VXLAN  │10.244.2.0│   VXLAN  │10.244.3.0│      │   │
│  │ └────┬─────┘          └────┬─────┘          └──────────┘      │   │
│  │      │                     │                                   │   │
│  │      └─────────────────────┘                                   │   │
│  │            Host Network                                         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                        Service Network                           │   │
│  │                                                                  │   │
│  │   Service IP (ClusterIP) ──────► kube-proxy ──────► Pod IPs     │   │
│  │   (10.96.0.0/12)                (iptables/ipvs)    (10.244.0.0) │   │
│  │                                                                  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 网络模型

| 网络类型 | CIDR | 说明 |
|----------|------|------|
| **Pod Network** | 10.244.0.0/16 | Pod IP 地址范围 |
| **Service Network** | 10.96.0.0/12 | Service ClusterIP 范围 |
| **Node Network** | 主机网络 | 节点间通信 |

### 2.3 CNI 集成

```go
// CNIManager CNI 管理器
type CNIManager struct {
    pluginDir string
    configDir string
    network   string
}

// CNIConfig CNI 配置
type CNIConfig struct {
    CNIVersion string   `json:"cniVersion"`
    Name       string   `json:"name"`
    Type       string   `json:"type"`
    Bridge     string   `json:"bridge"`
    IsGateway  bool     `json:"isGateway"`
    IPMasq     bool     `json:"ipMasq"`
    IPAM       IPAMConfig `json:"ipam"`
}

// IPAMConfig IPAM 配置
type IPAMConfig struct {
    Type    string   `json:"type"`
    Subnet  string   `json:"subnet"`
    Routes  []Route  `json:"routes"`
    Gateway string   `json:"gateway"`
}

// Route 路由配置
type Route struct {
    Dst string `json:"dst"`
}

// 默认 CNI 配置
const defaultCNIConfig = `{
    "cniVersion": "0.4.0",
    "name": "minik8s-net",
    "type": "bridge",
    "bridge": "cni0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.244.0.0/24",
        "routes": [
            { "dst": "0.0.0.0/0" }
        ],
        "gateway": "10.244.0.1"
    }
}`

// SetupPodNetwork 设置 Pod 网络
func (cm *CNIManager) SetupPodNetwork(podUID, namespace, name string, annotations map[string]string) (*NetworkResult, error) {
    // 生成容器 ID
    containerID := fmt.Sprintf("%s_%s_%s", namespace, name, podUID)
    
    // 创建网络命名空间路径
    netnsPath := fmt.Sprintf("/var/run/netns/%s", containerID)
    
    // 调用 CNI 插件
    result, err := cm.invokeCNI("ADD", containerID, netnsPath, annotations)
    if err != nil {
        return nil, fmt.Errorf("CNI ADD failed: %v", err)
    }
    
    return result, nil
}

// TeardownPodNetwork 清理 Pod 网络
func (cm *CNIManager) TeardownPodNetwork(podUID, namespace, name string) error {
    containerID := fmt.Sprintf("%s_%s_%s", namespace, name, podUID)
    netnsPath := fmt.Sprintf("/var/run/netns/%s", containerID)
    
    _, err := cm.invokeCNI("DEL", containerID, netnsPath, nil)
    if err != nil {
        return fmt.Errorf("CNI DEL failed: %v", err)
    }
    
    return nil
}

// invokeCNI 调用 CNI 插件
func (cm *CNIManager) invokeCNI(command, containerID, netnsPath string, annotations map[string]string) (*NetworkResult, error) {
    // 读取 CNI 配置
    config, err := cm.loadCNIConfig()
    if err != nil {
        return nil, err
    }
    
    // 设置环境变量
    env := []string{
        fmt.Sprintf("CNI_COMMAND=%s", command),
        fmt.Sprintf("CNI_CONTAINERID=%s", containerID),
        fmt.Sprintf("CNI_NETNS=%s", netnsPath),
        fmt.Sprintf("CNI_IFNAME=eth0"),
        fmt.Sprintf("CNI_PATH=%s", cm.pluginDir),
    }
    
    // 执行 CNI 插件
    cmd := exec.Command(filepath.Join(cm.pluginDir, config.Type))
    cmd.Env = env
    cmd.Stdin = strings.NewReader(cm.getCNIConfigJSON())
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("CNI plugin failed: %v, output: %s", err, output)
    }
    
    // 解析结果
    if command == "ADD" {
        var result NetworkResult
        if err := json.Unmarshal(output, &result); err != nil {
            return nil, err
        }
        return &result, nil
    }
    
    return nil, nil
}

// NetworkResult CNI 结果
type NetworkResult struct {
    CNIVersion string    `json:"cniVersion"`
    Interfaces []Interface `json:"interfaces,omitempty"`
    IPs        []IPConfig  `json:"ips,omitempty"`
    Routes     []RouteConfig `json:"routes,omitempty"`
    DNS        DNSConfig   `json:"dns,omitempty"`
}

type IPConfig struct {
    Interface int    `json:"interface"`
    Address   string `json:"address"`
    Gateway   string `json:"gateway,omitempty"`
}

type RouteConfig struct {
    Dst string `json:"dst"`
    GW  string `json:"gw,omitempty"`
}

type DNSConfig struct {
    Nameservers []string `json:"nameservers,omitempty"`
    Domain      string   `json:"domain,omitempty"`
    Search      []string `json:"search,omitempty"`
    Options     []string `json:"options,omitempty"`
}
```

### 2.4 Service 代理实现

```go
// ServiceProxy 服务代理
type ServiceProxy struct {
    iptables *iptables.IPTables
    services map[string]*ServiceInfo
    mu       sync.RWMutex
}

// ServiceInfo 服务信息
type ServiceInfo struct {
    Service   *Service
    Endpoints []string
}

// NewServiceProxy 创建服务代理
func NewServiceProxy() (*ServiceProxy, error) {
    ipt, err := iptables.New()
    if err != nil {
        return nil, err
    }
    
    return &ServiceProxy{
        iptables: ipt,
        services: make(map[string]*ServiceInfo),
    }, nil
}

// SyncServices 同步服务规则
func (sp *ServiceProxy) SyncServices(services []*Service, endpoints map[string]*Endpoints) error {
    sp.mu.Lock()
    defer sp.mu.Unlock()
    
    // 清理旧规则
    if err := sp.cleanupRules(); err != nil {
        return err
    }
    
    // 为每个服务创建规则
    for _, svc := range services {
        if svc.Spec.Type != ServiceTypeClusterIP && svc.Spec.Type != ServiceTypeNodePort {
            continue
        }
        
        key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
        eps := endpoints[key]
        
        if err := sp.setupServiceRules(svc, eps); err != nil {
            return fmt.Errorf("failed to setup rules for service %s: %v", key, err)
        }
        
        // 保存服务信息
        var endpointIPs []string
        for _, subset := range eps.Subsets {
            for _, addr := range subset.Addresses {
                endpointIPs = append(endpointIPs, addr.IP)
            }
        }
        
        sp.services[key] = &ServiceInfo{
            Service:   svc,
            Endpoints: endpointIPs,
        }
    }
    
    return nil
}

// setupServiceRules 设置服务规则
func (sp *ServiceProxy) setupServiceRules(svc *Service, eps *Endpoints) error {
    clusterIP := svc.Spec.ClusterIP
    if clusterIP == "None" || clusterIP == "" {
        return nil
    }
    
    for _, port := range svc.Spec.Ports {
        // 创建 ClusterIP 规则
        chainName := sp.serviceChainName(svc, port)
        
        // 创建服务链
        if err := sp.iptables.NewChain("nat", chainName); err != nil {
            return err
        }
        
        // 添加端点规则
        for _, subset := range eps.Subsets {
            for _, addr := range subset.Addresses {
                for _, epPort := range subset.Ports {
                    if epPort.Name == port.Name {
                        rule := []string{
                            "-d", clusterIP,
                            "-p", strings.ToLower(string(port.Protocol)),
                            "--dport", strconv.Itoa(int(port.Port)),
                            "-j", "DNAT",
                            "--to-destination", fmt.Sprintf("%s:%d", addr.IP, epPort.Port),
                        }
                        if err := sp.iptables.Append("nat", chainName, rule...); err != nil {
                            return err
                        }
                    }
                }
            }
        }
        
        // 跳转到服务链
        rule := []string{
            "-d", clusterIP,
            "-p", strings.ToLower(string(port.Protocol)),
            "--dport", strconv.Itoa(int(port.Port)),
            "-j", chainName,
        }
        if err := sp.iptables.Append("nat", "PREROUTING", rule...); err != nil {
            return err
        }
        if err := sp.iptables.Append("nat", "OUTPUT", rule...); err != nil {
            return err
        }
        
        // NodePort 规则
        if svc.Spec.Type == ServiceTypeNodePort && port.NodePort != 0 {
            nodePortRule := []string{
                "-p", strings.ToLower(string(port.Protocol)),
                "--dport", strconv.Itoa(int(port.NodePort)),
                "-j", chainName,
            }
            if err := sp.iptables.Append("nat", "PREROUTING", nodePortRule...); err != nil {
                return err
            }
        }
    }
    
    return nil
}

// cleanupRules 清理所有规则
func (sp *ServiceProxy) cleanupRules() error {
    // 清理自定义链
    chains, err := sp.iptables.ListChains("nat")
    if err != nil {
        return err
    }
    
    for _, chain := range chains {
        if strings.HasPrefix(chain, "MINIK8S-SVC-") {
            // 清空链
            sp.iptables.ClearChain("nat", chain)
            // 删除链
            sp.iptables.DeleteChain("nat", chain)
        }
    }
    
    // 清理 PREROUTING 规则
    rules, _ := sp.iptables.List("nat", "PREROUTING")
    for _, rule := range rules {
        if strings.Contains(rule, "MINIK8S-SVC-") {
            sp.iptables.Delete("nat", "PREROUTING", strings.Fields(rule)[2:]...)
        }
    }
    
    return nil
}

// serviceChainName 生成服务链名称
func (sp *ServiceProxy) serviceChainName(svc *Service, port ServicePort) string {
    hash := md5.Sum([]byte(fmt.Sprintf("%s/%s:%s", svc.Namespace, svc.Name, port.Name)))
    return fmt.Sprintf("MINIK8S-SVC-%s", hex.EncodeToString(hash[:8]))
}
```

### 2.5 DNS 服务

```go
// DNSResolver DNS 解析器
type DNSResolver struct {
    services  map[string]string  // service name -> clusterIP
    mu        sync.RWMutex
    upstream  string
}

// NewDNSResolver 创建 DNS 解析器
func NewDNSResolver(upstream string) *DNSResolver {
    return &DNSResolver{
        services: make(map[string]string),
        upstream: upstream,
    }
}

// UpdateServices 更新服务映射
func (d *DNSResolver) UpdateServices(services []*Service) {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    d.services = make(map[string]string)
    for _, svc := range services {
        if svc.Spec.ClusterIP == "None" {
            continue
        }
        
        // 添加短名称
        key := fmt.Sprintf("%s.%s", svc.Name, svc.Namespace)
        d.services[key] = svc.Spec.ClusterIP
        
        // 添加完整名称
        fullKey := fmt.Sprintf("%s.svc.cluster.local", key)
        d.services[fullKey] = svc.Spec.ClusterIP
    }
}

// Resolve 解析域名
func (d *DNSResolver) Resolve(name string) (string, bool) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    ip, ok := d.services[name]
    return ip, ok
}

// CoreDNS 配置模板
const coreDNSConfigTemplate = `.:53 {
    errors
    health
    ready
    
    # Kubernetes 插件配置
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
    }
    
    # 转发到上游 DNS
    forward . /etc/resolv.conf
    
    cache 30
    loop
    reload
    loadbalance
}
`
```

### 2.6 跨节点网络

```go
// VXLANManager VXLAN 管理器
type VXLANManager struct {
    deviceName string
    vni        int
    port       int
    localIP    string
    peers      map[string]string // node name -> node IP
    mu         sync.RWMutex
}

// NewVXLANManager 创建 VXLAN 管理器
func NewVXLANManager(localIP string, vni int) *VXLANManager {
    return &VXLANManager{
        deviceName: "vxlan.minik8s",
        vni:        vni,
        port:       4789,
        localIP:    localIP,
        peers:      make(map[string]string),
    }
}

// Setup 设置 VXLAN 设备
func (vm *VXLANManager) Setup() error {
    // 创建 VXLAN 设备
    link := &netlink.Vxlan{
        LinkAttrs: netlink.LinkAttrs{
            Name: vm.deviceName,
        },
        VxlanId: vni,
        Port:    port,
        Group:   net.ParseIP("0.0.0.0"),
        Local:   net.ParseIP(vm.localIP),
    }
    
    if err := netlink.LinkAdd(link); err != nil {
        return fmt.Errorf("failed to create vxlan device: %v", err)
    }
    
    // 启动设备
    if err := netlink.LinkSetUp(link); err != nil {
        return err
    }
    
    return nil
}

// AddPeer 添加对等节点
func (vm *VXLANManager) AddPeer(nodeName, nodeIP string, podCIDR string) error {
    vm.mu.Lock()
    defer vm.mu.Unlock()
    
    // 添加 FDB 条目
    dst := net.ParseIP(nodeIP)
    if dst == nil {
        return fmt.Errorf("invalid node IP: %s", nodeIP)
    }
    
    // 添加路由
    _, ipnet, err := net.ParseCIDR(podCIDR)
    if err != nil {
        return err
    }
    
    link, err := netlink.LinkByName(vm.deviceName)
    if err != nil {
        return err
    }
    
    route := &netlink.Route{
        LinkIndex: link.Attrs().Index,
        Dst:       ipnet,
        Gw:        dst,
    }
    
    if err := netlink.RouteAdd(route); err != nil {
        return fmt.Errorf("failed to add route: %v", err)
    }
    
    vm.peers[nodeName] = nodeIP
    return nil
}

// RemovePeer 移除对等节点
func (vm *VXLANManager) RemovePeer(nodeName string) error {
    vm.mu.Lock()
    defer vm.mu.Unlock()
    
    delete(vm.peers, nodeName)
    return nil
}
```

---

## 3. 跨平台网络支持

### 3.1 平台差异

| 平台 | 网络模式 | 限制 |
|------|----------|------|
| **Linux** | 完整 CNI 支持 | 无限制 |
| **macOS** | 用户态网络 | 需要虚拟机或 Docker Desktop |
| **Windows** | 用户态网络 | 需要 WSL2 或虚拟机 |
| **Android** | 受限模式 | 仅支持 host 网络 |

### 3.2 用户态网络（macOS/Windows）

```go
// UserspaceNetwork 用户态网络实现
type UserspaceNetwork struct {
    tapDevice string
    subnet    string
    gateway   string
}

// Setup 设置用户态网络
func (un *UserspaceNetwork) Setup() error {
    // 创建 TAP 设备
    // 使用 gvisor/netstack 或类似方案
    return nil
}

// ForwardPort 端口转发
func (un *UserspaceNetwork) ForwardPort(hostPort int, containerIP string, containerPort int) error {
    // 设置端口映射
    return nil
}
```

### 3.3 Android 受限模式

```go
// AndroidNetwork Android 网络实现
type AndroidNetwork struct {
    // Android 上使用 host 网络模式
    // 通过 proot/chroot 运行容器
}

// Setup 设置 Android 网络
func (an *AndroidNetwork) Setup() error {
    // 检查 root 权限
    // 配置网络命名空间
    return nil
}
```

---

## 4. 与 Kubernetes 的差异

| 特性 | Kubernetes | MiniK8s |
|------|------------|---------|
| 存储插件 | CSI 完整生态 | 内置基础类型 |
| 网络插件 | CNI 多插件支持 | bridge + host-local |
| 存储快照 | 支持 | 不支持 |
| 动态供应 | 支持 | 不支持 |
| 网络策略 | NetworkPolicy | 不支持 |
| Ingress | 多种控制器 | 不支持 |
| 负载均衡 | 多模式 | iptables DNAT |
| 跨节点网络 | 多种方案 | VXLAN 简化 |

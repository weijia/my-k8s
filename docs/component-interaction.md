# MiniK8s 组件交互设计文档

## 1. 组件交互概述

### 1.1 交互原则
- **松耦合**: 组件间通过定义良好的接口通信
- **异步优先**: 使用事件驱动机制减少阻塞
- **容错设计**: 组件故障不影响整体系统
- **可观测性**: 关键操作记录日志和事件

### 1.2 通信方式
| 通信方式 | 使用场景 | 协议 |
|----------|----------|------|
| HTTP REST | 外部 API 调用 | HTTP/1.1 |
| gRPC | 内部组件通信 | HTTP/2 |
| Watch/Event | 状态变更通知 | SSE/WebSocket |
| 共享存储 | 状态持久化 | SQLite |

---

## 2. 组件架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Control Plane                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │
│  │  API Server  │  │  Scheduler   │  │  Controller  │  │   Store    │  │
│  │              │◄─┤              │  │   Manager    │  │  (SQLite)  │  │
│  │  - REST API  │  │  - Filter    │  │              │  │            │  │
│  │  - Watch     │  │  - Score     │  │  - Replica   │  │  - State   │  │
│  │  - Validate  │  │  - Bind      │  │  - Service   │  │  - Events  │  │
│  └──────┬───────┘  └──────────────┘  └──────┬───────┘  └────────────┘  │
│         │                                    │                          │
│         └────────────────┬───────────────────┘                          │
│                          │                                              │
│                          ▼                                              │
│                   ┌──────────────┐                                      │
│                   │  Event Bus   │                                      │
│                   │  (Internal)  │                                      │
│                   └──────────────┘                                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTP/gRPC
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              Node Agent                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │
│  │   Kubelet    │  │   Runtime    │  │   CNI/Net    │  │   Volume   │  │
│  │              │  │              │  │              │  │            │  │
│  │  - Pod Mgr   │  │  - Container │  │  - IPAM      │  │  - Mount   │  │
│  │  - Probe     │  │  - Image     │  │  - Routes    │  │  - Cleanup │  │
│  │  - Metrics   │  │  - Exec      │  │  - DNS       │  │            │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 核心交互流程

### 3.1 Pod 创建流程

```
┌─────────┐     ┌───────────┐     ┌───────────┐     ┌───────────┐     ┌───────────┐
│  User   │     │ API Server│     │  Store    │     │ Scheduler │     │Node Agent │
└────┬────┘     └─────┬─────┘     └─────┬─────┘     └─────┬─────┘     └─────┬─────┘
     │                │                 │                 │                 │
     │ POST /pods     │                 │                 │                 │
     │───────────────►│                 │                 │                 │
     │                │                 │                 │                 │
     │                │ Save Pod (Pending)                │                 │
     │                │────────────────►│                 │                 │
     │                │                 │                 │                 │
     │                │ Publish Event   │                 │                 │
     │                │────────────────►│                 │                 │
     │                │                 │                 │                 │
     │                │                 │ Notify Scheduler│                 │
     │                │────────────────►│────────────────►│                 │
     │                │                 │                 │                 │
     │                │                 │                 │ Query Nodes     │
     │                │                 │                 │────────────────►│
     │                │                 │                 │                 │
     │                │                 │                 │◄────────────────│
     │                │                 │                 │                 │
     │                │                 │                 │ Filter & Score  │
     │                │                 │                 │─────────────────│
     │                │                 │                 │                 │
     │                │                 │                 │ Update Pod (nodeName)│
     │                │◄────────────────│◄────────────────│                 │
     │                │                 │                 │                 │
     │                │ Publish Event   │                 │                 │
     │                │────────────────►│                 │                 │
     │                │                 │                 │                 │
     │                │                 │ Notify Node Agent                 │
     │                │────────────────►│────────────────►│                 │
     │                │                 │                 │                 │
     │                │                 │                 │ Create Pod      │
     │                │                 │                 │────────────────►│
     │                │                 │                 │                 │
     │                │                 │                 │ Update Status   │
     │                │◄────────────────│◄────────────────│◄────────────────│
     │                │                 │                 │                 │
     │ 201 Created    │                 │                 │                 │
     │◄───────────────│                 │                 │                 │
     │                │                 │                 │                 │
```

**流程说明**:
1. 用户通过 API 提交 Pod 创建请求
2. API Server 验证并保存 Pod 到存储（状态为 Pending）
3. 发布 Pod 创建事件
4. Scheduler 监听到事件，执行调度算法
5. Scheduler 选择合适节点，更新 Pod 的 nodeName
6. 发布 Pod 调度事件
7. 对应 Node Agent 监听到事件，创建容器
8. Node Agent 更新 Pod 状态为 Running
9. API Server 返回创建结果给用户

---

### 3.2 Pod 删除流程

```
┌─────────┐     ┌───────────┐     ┌───────────┐     ┌───────────┐
│  User   │     │ API Server│     │  Store    │     │Node Agent │
└────┬────┘     └─────┬─────┘     └─────┬─────┘     └─────┬─────┘
     │                │                 │                 │
     │ DELETE /pods/x │                 │                 │
     │───────────────►│                 │                 │
     │                │                 │                 │
     │                │ Update (Terminating)              │
     │                │────────────────►│                 │
     │                │                 │                 │
     │                │ Publish Event   │                 │
     │                │────────────────►│                 │
     │                │                 │                 │
     │                │                 │ Notify Node     │
     │                │────────────────►│────────────────►│
     │                │                 │                 │
     │                │                 │                 │ Stop Containers
     │                │                 │                 │─────────────────
     │                │                 │                 │
     │                │                 │                 │ Cleanup
     │                │                 │                 │─────────────────
     │                │                 │                 │
     │                │                 │                 │ Delete Pod
     │                │◄────────────────│◄────────────────│
     │                │                 │                 │
     │ 200 OK         │                 │                 │
     │◄───────────────│                 │                 │
     │                │                 │                 │
```

---

### 3.3 Deployment 滚动更新流程

```
┌───────────┐     ┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│   User    │     │  API Server   │     │  Deployment   │     │  ReplicaSet   │
│           │     │               │     │  Controller   │     │  Controller   │
└─────┬─────┘     └───────┬───────┘     └───────┬───────┘     └───────┬───────┘
      │                   │                     │                     │
      │ PATCH /deployments│                     │                     │
      │ (image update)    │                     │                     │
      │──────────────────►│                     │                     │
      │                   │                     │                     │
      │                   │ Update Deployment   │                     │
      │                   │────────────────────►│                     │
      │                   │                     │                     │
      │                   │                     │ Create New RS       │
      │                   │────────────────────►│                     │
      │                   │                     │                     │
      │                   │                     │ Scale Up New RS     │
      │                   │                     │────────────────────►│
      │                   │                     │                     │
      │                   │                     │ Scale Down Old RS   │
      │                   │                     │────────────────────►│
      │                   │                     │                     │
      │                   │                     │ (repeat until done) │
      │                   │                     │                     │
      │ 200 OK            │                     │                     │
      │◄──────────────────│                     │                     │
      │                   │                     │                     │
```

---

### 3.4 Service 端点更新流程

```
┌───────────┐     ┌───────────────┐     ┌───────────────┐     ┌───────────┐
│   Pod     │     │  Node Agent   │     │  Service      │     │  kube-    │
│  Event    │     │               │     │  Controller   │     │  proxy    │
└─────┬─────┘     └───────┬───────┘     └───────┬───────┘     └─────┬─────┘
      │                   │                     │                   │
      │ Pod Created/Deleted/Updated              │                   │
      │──────────────────►│                     │                   │
      │                   │                     │                   │
      │                   │ Report Status       │                   │
      │                   │────────────────────►│                   │
      │                   │                     │                   │
      │                   │                     │ Select Matching   │
      │                   │                     │ Services          │
      │                   │                     │──────────────────►│
      │                   │                     │                   │
      │                   │                     │ Update Endpoints  │
      │                   │                     │──────────────────►│
      │                   │                     │                   │
      │                   │                     │ Sync iptables/ipvs│
      │                   │                     │──────────────────►│
      │                   │                     │                   │
```

---

## 4. 内部事件系统

### 4.1 事件类型定义

```go
// EventType 事件类型
type EventType string

const (
    // 资源生命周期事件
    EventCreated   EventType = "CREATED"
    EventUpdated   EventType = "UPDATED"
    EventDeleted   EventType = "DELETED"
    
    // 调度事件
    EventScheduled EventType = "SCHEDULED"
    EventFailedScheduling EventType = "FAILED_SCHEDULING"
    
    // 运行时事件
    EventContainerStarted  EventType = "CONTAINER_STARTED"
    EventContainerStopped  EventType = "CONTAINER_STOPPED"
    EventContainerFailed   EventType = "CONTAINER_FAILED"
    EventHealthCheckFailed EventType = "HEALTH_CHECK_FAILED"
    
    // 节点事件
    EventNodeReady    EventType = "NODE_READY"
    EventNodeNotReady EventType = "NODE_NOT_READY"
)

// Event 内部事件
type Event struct {
    Type      EventType
    Object    Object
    OldObject Object        // 用于更新事件
    Timestamp time.Time
}

// EventHandler 事件处理器
type EventHandler interface {
    OnCreate(obj Object)
    OnUpdate(oldObj, newObj Object)
    OnDelete(obj Object)
}
```

### 4.2 事件总线实现

```go
// EventBus 事件总线
type EventBus struct {
    handlers map[ResourceType][]EventHandler
    mu       sync.RWMutex
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(resourceType ResourceType, handler EventHandler) {
    eb.mu.Lock()
    defer eb.mu.Unlock()
    eb.handlers[resourceType] = append(eb.handlers[resourceType], handler)
}

// Publish 发布事件
func (eb *EventBus) Publish(event Event) {
    eb.mu.RLock()
    handlers := eb.handlers[getResourceType(event.Object)]
    eb.mu.RUnlock()
    
    for _, handler := range handlers {
        go eb.dispatch(event, handler)
    }
}

func (eb *EventBus) dispatch(event Event, handler EventHandler) {
    switch event.Type {
    case EventCreated:
        handler.OnCreate(event.Object)
    case EventUpdated:
        handler.OnUpdate(event.OldObject, event.Object)
    case EventDeleted:
        handler.OnDelete(event.Object)
    }
}
```

---

## 5. Watch 机制实现

### 5.1 Watch 接口

```go
// Watcher 资源监视器
type Watcher struct {
    resultChan chan WatchEvent
    stopChan   chan struct{}
    stopped    bool
    mu         sync.Mutex
}

// WatchEvent 监视事件
type WatchEvent struct {
    Type   EventType
    Object Object
}

// ResultChan 返回事件通道
func (w *Watcher) ResultChan() <-chan WatchEvent {
    return w.resultChan
}

// Stop 停止监视
func (w *Watcher) Stop() {
    w.mu.Lock()
    defer w.mu.Unlock()
    if !w.stopped {
        close(w.stopChan)
        w.stopped = true
    }
}
```

### 5.2 Watch 实现流程

```go
// Watch 监听资源变化
func (s *store) Watch(kind, namespace string) (Watcher, error) {
    watcher := &Watcher{
        resultChan: make(chan WatchEvent, 100),
        stopChan:   make(chan struct{}),
    }
    
    // 注册 watcher
    s.watchersMu.Lock()
    s.watchers[kind] = append(s.watchers[kind], watcher)
    s.watchersMu.Unlock()
    
    // 清理函数
    go func() {
        <-watcher.stopChan
        s.removeWatcher(kind, watcher)
    }()
    
    return watcher, nil
}

// notifyWatchers 通知所有 watcher
func (s *store) notifyWatchers(kind string, event WatchEvent) {
    s.watchersMu.RLock()
    watchers := s.watchers[kind]
    s.watchersMu.RUnlock()
    
    for _, watcher := range watchers {
        select {
        case watcher.resultChan <- event:
        case <-watcher.stopChan:
        default:
            // 通道满，丢弃事件
        }
    }
}
```

---

## 6. 组件间接口定义

### 6.1 API Server 接口

```go
// APIServer 接口
type APIServer interface {
    // 资源操作
    Create(ctx context.Context, obj Object) (Object, error)
    Get(ctx context.Context, kind, namespace, name string) (Object, error)
    Update(ctx context.Context, obj Object) (Object, error)
    Delete(ctx context.Context, kind, namespace, name string) error
    List(ctx context.Context, kind, namespace string, opts ListOptions) (List, error)
    Watch(ctx context.Context, kind, namespace string) (Watcher, error)
    
    // 健康检查
    Health() error
    Ready() error
}

// ListOptions 列表选项
type ListOptions struct {
    LabelSelector string
    FieldSelector string
    Limit         int64
    Continue      string
}
```

### 6.2 Scheduler 接口

```go
// Scheduler 调度器接口
type Scheduler interface {
    // Schedule 为 Pod 选择节点
    Schedule(ctx context.Context, pod *Pod) (string, error)
    
    // 预选阶段
    RunPreFilterPlugins(ctx context.Context, pod *Pod, nodes []*Node) ([]*Node, error)
    
    // 优选阶段
    RunFilterPlugins(ctx context.Context, pod *Pod, nodes []*Node) ([]*Node, error)
    RunScorePlugins(ctx context.Context, pod *Pod, nodes []*Node) (map[string]int64, error)
    
    // 绑定
    Bind(ctx context.Context, pod *Pod, nodeName string) error
}

// PreFilterPlugin 预选插件
type PreFilterPlugin interface {
    Name() string
    PreFilter(ctx context.Context, pod *Pod, nodes []*Node) ([]*Node, error)
}

// FilterPlugin 过滤插件
type FilterPlugin interface {
    Name() string
    Filter(ctx context.Context, pod *Pod, node *Node) bool
}

// ScorePlugin 打分插件
type ScorePlugin interface {
    Name() string
    Score(ctx context.Context, pod *Pod, node *Node) int64
}
```

### 6.3 Node Agent 接口

```go
// NodeAgent 节点代理接口
type NodeAgent interface {
    // Pod 生命周期
    CreatePod(ctx context.Context, pod *Pod) error
    UpdatePod(ctx context.Context, pod *Pod) error
    DeletePod(ctx context.Context, pod *Pod) error
    GetPodStatus(ctx context.Context, namespace, name string) (*PodStatus, error)
    
    // 容器操作
    ExecInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string) error
    GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts LogOptions) (io.ReadCloser, error)
    
    // 节点信息
    GetNodeStatus(ctx context.Context) (*NodeStatus, error)
    
    // 健康检查
    Health() error
}

// LogOptions 日志选项
type LogOptions struct {
    TailLines    *int64
    LimitBytes   *int64
    SinceSeconds *int64
    Follow       bool
}
```

### 6.4 Runtime 接口

```go
// Runtime 容器运行时接口
type Runtime interface {
    // 容器生命周期
    CreateContainer(ctx context.Context, config *ContainerConfig) (string, error)
    StartContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string, timeout int64) error
    RemoveContainer(ctx context.Context, containerID string) error
    
    // 容器状态
    GetContainerStatus(ctx context.Context, containerID string) (*ContainerStatus, error)
    ListContainers(ctx context.Context, filters ContainerFilters) ([]*Container, error)
    
    // 镜像管理
    PullImage(ctx context.Context, image string) error
    RemoveImage(ctx context.Context, image string) error
    
    // 执行和日志
    Exec(ctx context.Context, containerID string, cmd []string) error
    GetLogs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error)
}

// ContainerConfig 容器配置
type ContainerConfig struct {
    Name       string
    Image      string
    Command    []string
    Args       []string
    Env        map[string]string
    WorkingDir string
    
    // 资源限制
    CPUQuota   int64
    CPUPeriod  int64
    MemoryLimit int64
    
    // 网络
    NetworkMode string
    DNS         []string
    
    // 挂载
    Mounts []Mount
}

// Mount 挂载点
type Mount struct {
    Source      string
    Destination string
    Type        string
    ReadOnly    bool
}
```

---

## 7. 错误处理与重试机制

### 7.1 错误分类

```go
// ErrorType 错误类型
type ErrorType string

const (
    ErrorTypeNotFound      ErrorType = "NotFound"
    ErrorTypeAlreadyExists ErrorType = "AlreadyExists"
    ErrorTypeInvalid       ErrorType = "Invalid"
    ErrorTypeTimeout       ErrorType = "Timeout"
    ErrorTypeInternal      ErrorType = "Internal"
    ErrorTypeUnavailable   ErrorType = "Unavailable"
)

// APIError API 错误
type APIError struct {
    Type    ErrorType
    Message string
    Reason  string
    Details map[string]interface{}
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}
```

### 7.2 重试策略

```go
// RetryPolicy 重试策略
type RetryPolicy struct {
    MaxRetries  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

// DefaultRetryPolicy 默认重试策略
var DefaultRetryPolicy = &RetryPolicy{
    MaxRetries:   3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     5 * time.Second,
    Multiplier:   2.0,
}

// RetryWithPolicy 带策略的重试
func RetryWithPolicy(ctx context.Context, policy *RetryPolicy, fn func() error) error {
    var err error
    delay := policy.InitialDelay
    
    for i := 0; i <= policy.MaxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }
        
        // 不可重试的错误
        if !isRetryableError(err) {
            return err
        }
        
        if i < policy.MaxRetries {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
                delay = time.Duration(float64(delay) * policy.Multiplier)
                if delay > policy.MaxDelay {
                    delay = policy.MaxDelay
                }
            }
        }
    }
    
    return err
}

func isRetryableError(err error) bool {
    if apiErr, ok := err.(*APIError); ok {
        switch apiErr.Type {
        case ErrorTypeTimeout, ErrorTypeUnavailable:
            return true
        default:
            return false
        }
    }
    return true
}
```

---

## 8. 健康检查与故障恢复

### 8.1 组件健康检查

```go
// HealthChecker 健康检查器
type HealthChecker struct {
    components map[string]HealthCheckable
}

// HealthCheckable 可健康检查接口
type HealthCheckable interface {
    Health() error
    Ready() error
}

// CheckAll 检查所有组件
func (hc *HealthChecker) CheckAll() map[string]error {
    results := make(map[string]error)
    for name, component := range hc.components {
        results[name] = component.Health()
    }
    return results
}

// CheckReady 检查就绪状态
func (hc *HealthChecker) CheckReady() map[string]error {
    results := make(map[string]error)
    for name, component := range hc.components {
        results[name] = component.Ready()
    }
    return results
}
```

### 8.2 故障恢复策略

| 故障场景 | 检测方式 | 恢复策略 |
|----------|----------|----------|
| Node Agent 失联 | 心跳超时 | 标记节点 NotReady，重新调度 Pod |
| 容器崩溃 | 退出码非0 | 根据重启策略重启容器 |
| 健康检查失败 | Probe 失败 | 重启容器或标记未就绪 |
| 镜像拉取失败 | 拉取超时 | 指数退避重试 |
| API Server 不可用 | 连接失败 | 客户端重试，缓存操作 |

---

## 9. 性能优化策略

### 9.1 缓存策略

```go
// Cache 资源缓存
type Cache struct {
    store map[string]Object
    mu    sync.RWMutex
}

// Get 从缓存获取
func (c *Cache) Get(key string) (Object, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    obj, ok := c.store[key]
    return obj, ok
}

// Set 设置缓存
func (c *Cache) Set(key string, obj Object) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.store[key] = obj
}

// Invalidate 使缓存失效
func (c *Cache) Invalidate(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    delete(c.store, key)
}
```

### 9.2 批处理优化

```go
// Batcher 批处理器
type Batcher struct {
    buffer   []Operation
    maxSize  int
    interval time.Duration
    handler  func([]Operation)
}

// Add 添加操作
func (b *Batcher) Add(op Operation) {
    b.mu.Lock()
    b.buffer = append(b.buffer, op)
    shouldFlush := len(b.buffer) >= b.maxSize
    b.mu.Unlock()
    
    if shouldFlush {
        b.Flush()
    }
}

// Flush 刷新缓冲区
func (b *Batcher) Flush() {
    b.mu.Lock()
    if len(b.buffer) == 0 {
        b.mu.Unlock()
        return
    }
    ops := b.buffer
    b.buffer = make([]Operation, 0, b.maxSize)
    b.mu.Unlock()
    
    b.handler(ops)
}
```

---

## 10. 与 Kubernetes 的差异

| 特性 | Kubernetes | MiniK8s |
|------|------------|---------|
| 组件通信 | etcd + 多 Master | SQLite + 单 Master |
| 事件系统 | etcd Watch | 内存事件总线 |
| 调度器 | 可扩展插件体系 | 内置简化算法 |
| 控制器 | 独立进程 | 协程内运行 |
| 节点通信 | gRPC 双向流 | HTTP 轮询 |
| 重试机制 | 完整退避策略 | 简化重试 |
| 缓存 | 多级缓存 | 内存缓存 |

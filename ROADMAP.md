# MiniK8s 开发路线图 (Roadmap)

## 项目概述

MiniK8s 是一个轻量级容器编排工具，与 Kubernetes 核心概念兼容，适用于资源受限环境。本文档详细规划了项目的分阶段实现计划。

---

## 开发阶段总览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           MiniK8s Roadmap                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Phase 1: 核心基础 (MVP)          Phase 2: 功能完善                          │
│  ├─ 项目脚手架                      ├─ ReplicaSet & Deployment               │
│  ├─ API Server + SQLite             ├─ Service & 基础网络                   │
│  ├─ Pod 生命周期管理                ├─ 调度器实现                           │
│  ├─ 容器运行时集成                  ├─ ConfigMap & Secret                   │
│  └─ 命令行工具                      └─ 健康检查 & 重启策略                  │
│       ▼                                    ▼                                 │
│       ╔═══════════════════════════════════════╗                             │
│       ║     Phase 3: 多平台支持               ║                             │
│       ║  ├─ Windows 支持                      ║                             │
│       ║  ├─ macOS 支持                        ║                             │
│       ║  ├─ Android CLI 支持                  ║                             │
│       ║  └─ Android APK 原型                  ║                             │
│       ╚═══════════════════════════════════════╝                             │
│                        ▼                                                     │
│  Phase 4: 优化和扩展                                                         │
│  ├─ 性能优化                                                                 │
│  ├─ 更多存储类型                                                             │
│  ├─ 监控和日志                                                               │
│  └─ 文档完善                                                                 │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: 核心基础 (MVP) - 预计 4-6 周

### 目标
建立项目基础架构，实现最基本的 Pod 管理能力。

### 1.1 项目脚手架 (Week 1)

#### 任务清单
- [ ] 初始化 Go 模块和项目结构
- [ ] 配置 Makefile 和构建脚本
- [ ] 设置代码规范和 lint 工具
- [ ] 创建基础 CI/CD 流程
- [ ] 编写开发环境搭建文档

#### 交付物
```
my-k8s/
├── cmd/
│   ├── minik8s/          # 主程序
│   └── kubectl/          # CLI 工具
├── pkg/
│   ├── api/              # API 类型定义
│   ├── storage/          # 存储层
│   └── utils/            # 工具函数
├── Makefile
├── go.mod
└── README.md
```

#### 技术要点
- Go 1.21+ 版本
- 使用 Go modules 管理依赖
- 配置 golangci-lint 代码检查
- GitHub Actions CI 流程

---

### 1.2 API Server + SQLite 存储 (Week 1-2)

#### 任务清单
- [ ] 实现基础 HTTP 服务器 (Gin/Echo)
- [ ] 定义核心资源类型 (Pod, Node)
- [ ] 实现 SQLite 存储层
- [ ] 实现资源 CRUD API
- [ ] 实现 Watch 机制基础

#### API 端点
```
POST   /api/v1/namespaces/{namespace}/pods
GET    /api/v1/namespaces/{namespace}/pods
GET    /api/v1/namespaces/{namespace}/pods/{name}
DELETE /api/v1/namespaces/{namespace}/pods/{name}
GET    /api/v1/nodes
```

#### 数据表结构
```sql
CREATE TABLE resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_version TEXT NOT NULL,
    kind TEXT NOT NULL,
    namespace TEXT,
    name TEXT NOT NULL,
    uid TEXT UNIQUE NOT NULL,
    data JSON NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace, name, kind)
);
```

---

### 1.3 容器运行时集成 (Week 2-3)

#### 任务清单
- [ ] 定义容器运行时接口 (CRI)
- [ ] 集成 containerd 客户端
- [ ] 实现镜像拉取功能
- [ ] 实现容器生命周期管理
- [ ] 实现容器日志获取

#### CRI 接口定义
```go
type Runtime interface {
    PullImage(ctx context.Context, image string) error
    CreateContainer(ctx context.Context, config *ContainerConfig) (string, error)
    StartContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string, timeout int64) error
    RemoveContainer(ctx context.Context, containerID string) error
    GetContainerStatus(ctx context.Context, containerID string) (*ContainerStatus, error)
    GetContainerLogs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error)
}
```

#### 依赖安装
```bash
# Ubuntu/Debian
apt-get install containerd

# 或使用 nerdctl
```

---

### 1.4 Pod 生命周期管理 (Week 3-4)

#### 任务清单
- [ ] 实现 Pod 创建流程
- [ ] 实现 Pod 状态管理
- [ ] 实现 Pod 删除流程
- [ ] 实现 Pod 列表查询
- [ ] 实现基础事件记录

#### Pod 状态流转
```
Pending -> Running -> Succeeded
   |          |
   |          -> Failed
   -> Failed
```

#### 核心功能
- 解析 Pod Spec，创建对应容器
- 管理 Pause 容器（网络命名空间）
- 同步 Pod 状态到存储
- 处理容器退出码

---

### 1.5 命令行工具 (Week 4-5)

#### 任务清单
- [ ] 实现 kubectl 基础命令
- [ ] 实现资源创建/删除/查询
- [ ] 实现日志查看命令
- [ ] 实现 exec 进入容器
- [ ] 配置文件支持

#### 支持的命令
```bash
# 集群管理
minik8s start
minik8s stop
minik8s status

# 资源操作
kubectl create -f pod.yaml
kubectl get pods
kubectl get pods -o yaml
kubectl describe pod <name>
kubectl delete pod <name>
kubectl logs <pod-name>
kubectl exec <pod-name> -- <command>
```

---

### 1.6 Phase 1 验收标准

| 验收项 | 标准 |
|--------|------|
| 功能 | 可以创建、查看、删除 Pod |
| 稳定性 | 连续运行 24 小时无崩溃 |
| 性能 | 创建 Pod < 10 秒 |
| 测试 | 核心功能单元测试覆盖 > 60% |
| 文档 | API 文档和快速开始指南 |

---

## Phase 2: 功能完善 - 预计 6-8 周

### 目标
实现 Kubernetes 核心功能，包括调度、服务发现、配置管理。

### 2.1 ReplicaSet 控制器 (Week 1-2)

#### 任务清单
- [ ] 实现 ReplicaSet 资源类型
- [ ] 实现 ReplicaSet 控制器
- [ ] 实现 Pod 副本管理
- [ ] 实现标签选择器
- [ ] 实现 OwnerReference

#### 核心逻辑
```go
func (r *ReplicaSetController) reconcile(rs *ReplicaSet) error {
    // 获取当前 Pod 列表
    pods := r.getPodsBySelector(rs.Spec.Selector)
    
    // 计算差异
    diff := rs.Spec.Replicas - len(pods)
    
    if diff > 0 {
        // 创建新 Pod
        return r.createPods(rs, diff)
    } else if diff < 0 {
        // 删除多余 Pod
        return r.deletePods(pods, -diff)
    }
    
    return nil
}
```

---

### 2.2 Deployment 控制器 (Week 2-3)

#### 任务清单
- [ ] 实现 Deployment 资源类型
- [ ] 实现 Deployment 控制器
- [ ] 实现滚动更新策略
- [ ] 实现版本回滚基础
- [ ] 实现扩缩容功能

#### 滚动更新流程
```
1. 创建新 ReplicaSet
2. 逐步增加新 RS 副本数
3. 逐步减少旧 RS 副本数
4. 更新 Deployment 状态
```

---

### 2.3 Service 和基础网络 (Week 3-5)

#### 任务清单
- [ ] 实现 Service 资源类型
- [ ] 实现 Endpoints 管理
- [ ] 集成 CNI 插件
- [ ] 实现 ClusterIP 功能
- [ ] 实现 NodePort 功能
- [ ] 实现 kube-proxy 基础

#### 网络架构
```
Pod (10.244.x.x) -> cni0 bridge -> host network
                      |
Service (10.96.x.x) -> iptables DNAT -> Pod
```

#### CNI 配置
```json
{
    "cniVersion": "0.4.0",
    "name": "minik8s-net",
    "type": "bridge",
    "bridge": "cni0",
    "ipam": {
        "type": "host-local",
        "subnet": "10.244.0.0/24"
    }
}
```

---

### 2.4 调度器实现 (Week 5-6)

#### 任务清单
- [ ] 实现调度器框架
- [ ] 实现预选算法
- [ ] 实现优选算法
- [ ] 实现资源匹配
- [ ] 实现节点亲和性基础

#### 调度流程
```go
func (s *Scheduler) Schedule(pod *Pod) (string, error) {
    // 1. 获取所有节点
    nodes := s.getNodes()
    
    // 2. 预选：过滤不满足条件的节点
    filtered := s.preFilter(pod, nodes)
    
    // 3. 优选：为节点打分
    scores := s.score(pod, filtered)
    
    // 4. 选择最高分节点
    selected := s.selectBest(scores)
    
    return selected.Name, nil
}
```

#### 预选策略
- PodFitsResources: 资源充足性检查
- PodFitsHost: 节点名称匹配
- PodFitsHostPorts: 端口冲突检查

#### 优选策略
- LeastAllocated: 选择资源利用率最低的节点
- BalancedAllocation: 平衡 CPU 和内存使用

---

### 2.5 ConfigMap 和 Secret (Week 6)

#### 任务清单
- [ ] 实现 ConfigMap 资源类型
- [ ] 实现 Secret 资源类型
- [ ] 实现 Volume 挂载
- [ ] 实现环境变量注入
- [ ] 实现配置热更新

#### 使用方式
```yaml
# Volume 挂载
volumes:
- name: config
  configMap:
    name: app-config

# 环境变量注入
env:
- name: DB_HOST
  valueFrom:
    configMapKeyRef:
      name: app-config
      key: db.host
```

---

### 2.6 健康检查和重启策略 (Week 7-8)

#### 任务清单
- [ ] 实现 LivenessProbe
- [ ] 实现 ReadinessProbe
- [ ] 实现 HTTP/TCP/Exec 探针
- [ ] 实现重启策略
- [ ] 实现优雅终止

#### 探针类型
```go
type Probe struct {
    HTTPGet             *HTTPGetAction
    TCPSocket           *TCPSocketAction
    Exec                *ExecAction
    InitialDelaySeconds int32
    PeriodSeconds       int32
    TimeoutSeconds      int32
    FailureThreshold    int32
}
```

#### 重启策略
- Always: 始终重启（默认）
- OnFailure: 失败时重启
- Never: 不重启

---

### 2.7 Phase 2 验收标准

| 验收项 | 标准 |
|--------|------|
| 功能 | 支持 Deployment、Service、ConfigMap |
| 调度 | Pod 可以调度到不同节点 |
| 网络 | Pod 间可以相互访问 |
| 稳定性 | 滚动更新不中断服务 |
| 测试 | 集成测试覆盖主要功能 |

---

## Phase 3: 多平台支持 - 预计 4-6 周

### 目标
支持 Windows、macOS、Android 等多平台运行。

### 3.1 Windows 支持 (Week 1-2)

#### 任务清单
- [ ] 适配 Windows 文件路径
- [ ] 适配 Windows 进程管理
- [ ] 支持 Windows 容器（可选）
- [ ] 使用 WSL2 作为运行时
- [ ] 测试和文档

#### 技术方案
- 优先支持 WSL2 后端
- 使用 Docker Desktop 作为容器运行时
- 路径转换处理

---

### 3.2 macOS 支持 (Week 2-3)

#### 任务清单
- [ ] 适配 macOS 系统调用
- [ ] 集成 Docker Desktop
- [ ] 实现用户态网络方案
- [ ] 端口转发支持
- [ ] 测试和文档

#### 技术方案
- 使用 Docker Desktop 的 Kubernetes 后端
- 或实现基于 VM 的方案

---

### 3.3 Android CLI 支持 (Week 3-4)

#### 任务清单
- [ ] 交叉编译 Android 二进制
- [ ] 适配 Termux 环境
- [ ] 实现 rootless 容器模式
- [ ] 受限网络支持
- [ ] 测试和文档

#### 技术方案
```bash
# 交叉编译
GOOS=linux GOARCH=arm64 go build

# Termux 安装
pkg install minik8s
```

---

### 3.4 Android APK 原型 (Week 4-6)

#### 任务清单
- [ ] 使用 gomobile 构建
- [ ] 设计 Android UI
- [ ] 实现基础功能封装
- [ ] JNI 桥接层
- [ ] 发布测试 APK

#### 功能范围
- 查看 Pod 列表
- 创建简单 Pod
- 查看日志
- 基础监控

---

### 3.5 Phase 3 验收标准

| 平台 | 支持程度 | 验证方式 |
|------|----------|----------|
| Linux | 完整功能 | 原生运行 |
| Windows | WSL2 后端 | Docker Desktop |
| macOS | Docker 后端 | Docker Desktop |
| Android CLI | 受限功能 | Termux |
| Android APK | 基础功能 | 真机测试 |

---

## Phase 4: 优化和扩展 - 预计 4-6 周

### 目标
提升性能、可观测性，完善文档和生态。

### 4.1 性能优化 (Week 1-2)

#### 任务清单
- [ ] 实现内存缓存层
- [ ] 优化 Watch 机制
- [ ] 实现连接池
- [ ] 减少内存分配
- [ ] 性能基准测试

#### 优化目标
| 指标 | 目标 |
|------|------|
| API 延迟 | P99 < 100ms |
| 内存占用 | < 100MB |
| Pod 启动时间 | < 5 秒 |
| 并发处理 | 1000+ QPS |

---

### 4.2 监控和日志 (Week 2-3)

#### 任务清单
- [ ] 集成 Prometheus 指标
- [ ] 实现结构化日志
- [ ] 实现日志轮转
- [ ] 集成 Grafana 仪表盘
- [ ] 实现事件审计

#### 监控指标
```
minik8s_pods_total
minik8s_pod_status{status="Running"}
minik8s_container_restarts_total
minik8s_api_requests_total
minik8s_scheduler_latency_seconds
```

---

### 4.3 更多存储类型 (Week 3-4)

#### 任务清单
- [ ] 实现本地持久卷
- [ ] 实现 NFS 客户端
- [ ] 实现存储快照
- [ ] 存储容量跟踪
- [ ] 动态供应基础

---

### 4.4 文档完善 (Week 4-6)

#### 任务清单
- [ ] 编写完整用户指南
- [ ] 编写 API 参考文档
- [ ] 编写开发贡献指南
- [ ] 制作视频教程
- [ ] 编写最佳实践

#### 文档结构
```
docs/
├── getting-started/
│   ├── installation.md
│   ├── quickstart.md
│   └── configuration.md
├── user-guide/
│   ├── pods.md
│   ├── deployments.md
│   ├── services.md
│   └── networking.md
├── developer-guide/
│   ├── architecture.md
│   ├── building.md
│   └── testing.md
├── api-reference/
│   └── api.md
└── faq.md
```

---

### 4.5 Phase 4 验收标准

| 验收项 | 标准 |
|--------|------|
| 性能 | 达到优化目标 |
| 监控 | 提供完整仪表盘 |
| 文档 | 覆盖所有功能 |
| 稳定性 | 生产环境可用 |

---

## 里程碑和发布计划

### 版本规划

| 版本 | 内容 | 预计时间 |
|------|------|----------|
| v0.1.0 | Phase 1 完成，MVP 可用 | Month 1-2 |
| v0.2.0 | Phase 2 完成，核心功能 | Month 3-4 |
| v0.3.0 | Phase 3 完成，多平台 | Month 5-6 |
| v0.4.0 | Phase 4 完成，生产就绪 | Month 7-8 |
| v1.0.0 | 稳定版本 | Month 9+ |

### 发布检查清单

- [ ] 所有测试通过
- [ ] 文档已更新
- [ ] CHANGELOG 已编写
- [ ] 二进制已构建
- [ ] Docker 镜像已推送
- [ ] GitHub Release 已创建

---

## 开发规范

### 代码规范
- 遵循 Go Code Review Comments
- 使用 golangci-lint 检查
- 单元测试覆盖率 > 60%
- 核心功能必须有集成测试

### Git 工作流
- 使用 GitHub Flow
- PR 必须经过 Code Review
- CI 通过才能合并
- 提交信息遵循 Conventional Commits

### 版本控制
- 语义化版本 (SemVer)
- 主版本：不兼容 API 变更
- 次版本：向后兼容功能添加
- 修订版本：向后兼容问题修复

---

## 贡献指南

### 如何贡献
1. Fork 仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交变更 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 报告问题
- 使用 GitHub Issues
- 提供复现步骤
- 提供环境信息
- 提供日志输出

### 功能请求
- 先搜索是否已存在
- 描述使用场景
- 说明期望行为

---

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| containerd 集成复杂 | 高 | 先使用 Docker 作为备选 |
| CNI 跨平台问题 | 中 | 提供平台特定实现 |
| Android 限制 | 高 | 明确功能限制，提供替代方案 |
| 资源竞争 | 中 | 充分测试，优雅降级 |
| 进度延迟 | 中 | 分阶段交付，优先核心功能 |

---

## 资源需求

### 开发环境
- Go 1.21+
- Docker / containerd
- Linux 开发机（推荐 Ubuntu 22.04）
- 8GB+ 内存

### 测试环境
- Linux 虚拟机集群
- Windows 测试机
- macOS 测试机
- Android 设备

### 工具链
- IDE: VSCode / GoLand
- 调试: delve
- 测试: gotestsum
- 构建: make, docker

---

## 附录

### 参考资源
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [containerd Documentation](https://containerd.io/docs/)
- [CNI Specification](https://www.cni.dev/docs/spec/)
- [Go Best Practices](https://golang.org/doc/effective_go)

### 相关项目
- [k3s](https://k3s.io/) - 轻量级 Kubernetes
- [kind](https://kind.sigs.k8s.io/) - Kubernetes in Docker
- [minikube](https://minikube.sigs.k8s.io/) - 本地 Kubernetes

---

*最后更新: 2026-05-14*
*维护者: MiniK8s Team*

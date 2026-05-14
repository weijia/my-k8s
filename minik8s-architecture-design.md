# MiniK8s 架构设计文档

## 1. 项目概述

### 1.1 项目定位
MiniK8s 是一个轻量级容器编排工具，与 Kubernetes 核心概念兼容，但实现精简，适用于资源受限环境（边缘计算、IoT设备、开发测试环境）。项目兼具学习教学价值和轻量级生产用途。

### 1.2 设计原则
- **兼容性**: 与 Kubernetes API 概念兼容，降低学习成本
- **轻量级**: 单二进制文件，低资源占用（目标 < 100MB 内存）
- **跨平台**: 支持 Linux、macOS、Windows、Android (CLI + APK)
- **简化设计**: 去除严格隔离和复杂权限控制
- **扩展性**: 模块化架构，便于功能扩展

### 1.3 目标平台
| 平台 | 支持形式 | 说明 |
|------|----------|------|
| Linux | 原生二进制 | 主要目标平台，完整功能 |
| macOS | 原生二进制 | 开发测试环境 |
| Windows | 原生二进制 | 开发测试环境 |
| Android | CLI + APK | Termux CLI 和原生 Android App |

---

## 2. 整体架构

### 2.1 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        MiniK8s Architecture                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │   API Server    │◄──►│   Scheduler     │    │   Controller │  │
│  │   (HTTP/gRPC)   │    │   (调度器)       │    │   Manager    │  │
│  └────────┬────────┘    └─────────────────┘    └─────────────┘  │
│           │                                                      │
│           ▼                                                      │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │   State Store   │◄──►│   Node Agent    │◄──►│   Runtime   │  │
│  │   (SQLite/内存)  │    │   (Kubelet简化)  │    │  (Containerd│  │
│  └─────────────────┘    └─────────────────┘    │  /Docker)    │  │
│                                                 └─────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Networking Layer                       │   │
│  │         (CNI简化 / 内置网络 / 用户态网络)                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 组件说明

#### 2.2.1 API Server (API服务器)
- **职责**: 提供 RESTful API 接口，处理所有资源操作请求
- **实现**: 基于 Go 的 HTTP 服务器
- **存储**: 支持 SQLite (默认) 或内存存储
- **功能**:
  - 资源 CRUD 操作
  - Watch 机制（长轮询实现）
  - 简单的认证（可选）

#### 2.2.2 Scheduler (调度器)
- **职责**: 将 Pod 分配到合适的节点
- **算法**: 简化版调度算法
  - 过滤阶段：资源充足性检查
  - 打分阶段：资源利用率均衡
- **策略**: 支持节点亲和性、资源需求匹配

#### 2.2.3 Controller Manager (控制器管理器)
- **职责**: 维护集群期望状态
- **内置控制器**:
  - ReplicaSet Controller: 维护 Pod 副本数
  - Service Controller: 管理服务端点
  - Node Controller: 节点健康检查

#### 2.2.4 Node Agent (节点代理)
- **职责**: 节点上的 Pod 生命周期管理
- **功能**:
  - Pod 创建/停止/删除
  - 容器运行时交互
  - 资源监控和上报
  - 健康检查执行

#### 2.2.5 Runtime (容器运行时)
- **支持**: Containerd (推荐) / Docker
- **抽象层**: 统一运行时接口

#### 2.2.6 Networking (网络层)
- **实现选项**:
  - Linux: CNI 插件支持
  - 跨平台: 内置用户态网络（基于 gvisor/netstack 或类似方案）
  - Android: 受限网络模式

---

## 3. 支持的核心功能

### 3.1 资源对象

| 资源类型 | 支持程度 | 说明 |
|----------|----------|------|
| **Pod** | 完整支持 | 基本调度单位，支持多容器 |
| **ReplicaSet** | 完整支持 | Pod 副本管理 |
| **Deployment** | 基础支持 | 滚动更新（简化版） |
| **Service** | 基础支持 | ClusterIP、NodePort |
| **ConfigMap** | 完整支持 | 配置数据管理 |
| **Secret** | 基础支持 | 敏感数据（Base64，不加密） |
| **Node** | 完整支持 | 集群节点管理 |
| **Namespace** | 基础支持 | 资源隔离（逻辑隔离） |

### 3.2 功能特性

#### 3.2.1 Pod 生命周期
- [x] Pod 创建、运行、停止、删除
- [x] 容器镜像拉取
- [x] 重启策略（Always、OnFailure、Never）
- [x] 健康检查（Liveness/Readiness Probe）
- [x] 资源限制（CPU、内存）
- [x] 环境变量和命令行参数
- [x] Volume 挂载（hostPath、emptyDir）

#### 3.2.2 调度功能
- [x] 基础资源调度（CPU、内存匹配）
- [x] 节点标签选择器
- [x] Pod 亲和性/反亲和性（简化）
- [x] 资源配额检查

#### 3.2.3 网络功能
- [x] Pod IP 分配
- [x] Service ClusterIP（内部负载均衡）
- [x] Service NodePort（外部访问）
- [x] 基础 DNS 解析
- [ ] Ingress（未来扩展）

#### 3.2.4 存储功能
- [x] emptyDir（临时存储）
- [x] hostPath（主机目录挂载）
- [ ] PersistentVolume（未来扩展）

---

## 4. 技术选型

### 4.1 核心技术栈

| 组件 | 技术选型 | 理由 |
|------|----------|------|
| **编程语言** | Go 1.21+ | 云原生生态标准，跨平台编译 |
| **HTTP框架** | Gin / Echo | 轻量高效，中间件丰富 |
| **gRPC** | google.golang.org/grpc | 内部组件通信 |
| **数据库** | SQLite / 内存 | 零配置，嵌入式 |
| **容器运行时** | Containerd | 行业标准，轻量 |
| **网络** | CNI / 内置网络 | 灵活适配多平台 |

### 4.2 跨平台支持策略

#### 4.2.1 构建策略
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build

# Linux ARM64 (边缘设备)
GOOS=linux GOARCH=arm64 go build

# macOS
GOOS=darwin GOARCH=amd64 go build
GOOS=darwin GOARCH=arm64 go build

# Windows
GOOS=windows GOARCH=amd64 go build

# Android
# CLI: 通过 Termux 运行标准 Linux 二进制
# APK: 使用 gomobile 构建 Android 应用
```

#### 4.2.2 Android 特殊处理

**CLI 模式（Termux）**:
- 编译为 Linux ARM64 二进制
- 依赖 proot 或 chroot 运行容器（受限）
- 或使用 containerd 的 rootless 模式

**APK 模式（原生应用）**:
- 使用 gomobile 构建
- 嵌入 WebView 提供 GUI
- 通过 JNI 调用底层功能
- 受限功能（无完整容器支持，可用进程模拟）

---

## 5. 项目结构

```
minik8s/
├── cmd/
│   ├── minik8s/          # 主程序入口
│   ├── kubectl/          # 命令行客户端
│   └── node-agent/       # 节点代理（可选独立运行）
├── pkg/
│   ├── api/              # API 定义和序列化
│   │   ├── types/        # 资源类型定义
│   │   └── server/       # API Server 实现
│   ├── scheduler/        # 调度器
│   ├── controller/       # 控制器
│   ├── runtime/          # 容器运行时接口
│   ├── network/          # 网络管理
│   ├── storage/          # 状态存储
│   ├── node/             # 节点管理
│   └── utils/            # 工具函数
├── internal/             # 内部实现细节
├── configs/              # 配置文件示例
├── deployments/          # 部署脚本
├── docs/                 # 文档
├── android/              # Android 应用代码
│   ├── app/              # Android Studio 项目
│   └── bridge/           # Go-Java 桥接代码
├── go.mod
├── Makefile
└── README.md
```

---

## 6. 关键设计决策

### 6.1 简化设计点

| 方面 | Kubernetes | MiniK8s |
|------|------------|---------|
| **认证授权** | RBAC、Webhook | 可选简单token或无认证 |
| **网络隔离** | NetworkPolicy | 无隔离，或基础隔离 |
| **存储** | CSI 插件体系 | 内置基础存储类型 |
| **多租户** | Namespace 隔离 | 仅逻辑隔离 |
| **高可用** | 多 Master | 单 Master（可扩展） |
| **etcd** | 独立集群 | SQLite 或内存 |

### 6.2 不支持的 Kubernetes 特性

- RBAC 细粒度权限控制
- NetworkPolicy 网络策略
- PodSecurityPolicy / PodSecurity
- CustomResourceDefinition (CRD)
- Admission Webhook
- 高级调度（拓扑分布、优先级抢占）
- 自动扩缩容（HPA/VPA）
- 存储快照和动态供应

---

## 7. API 兼容性

### 7.1 兼容策略
- **概念兼容**: 使用相同的资源模型和术语
- **API 子集**: 支持核心 API 的子集
- **版本**: 使用 apiVersion: v1（简化）

### 7.2 示例资源定义

```yaml
# Pod 示例
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
  labels:
    app: nginx
spec:
  containers:
  - name: nginx
    image: nginx:alpine
    ports:
    - containerPort: 80
    resources:
      limits:
        memory: "128Mi"
        cpu: "500m"
    livenessProbe:
      httpGet:
        path: /
        port: 80
```

```yaml
# Service 示例
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
  type: NodePort
```

```yaml
# Deployment 示例
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
```

---

## 8. 部署模式

### 8.1 单节点模式（默认）
```
┌─────────────────────────────────────┐
│           Single Node                │
│  ┌─────────┐ ┌─────────┐ ┌────────┐ │
│  │API Server│ │Scheduler│ │Controller│
│  └────┬────┘ └─────────┘ └────────┘ │
│       │                              │
│  ┌────┴──────────────────────────┐   │
│  │         Node Agent             │   │
│  │  ┌─────────────────────────┐   │   │
│  │  │    Container Runtime    │   │   │
│  │  └─────────────────────────┘   │   │
│  └────────────────────────────────┘   │
└─────────────────────────────────────┘
```

### 8.2 多节点模式
```
┌─────────────────┐         ┌─────────────────┐
│   Master Node   │◄───────►│   Worker Node   │
│  ┌───────────┐  │         │  ┌───────────┐  │
│  │ API Server│  │         │  │Node Agent │  │
│  │ Scheduler │  │   HTTP  │  │  + Runtime│  │
│  │ Controller│  │◄───────►│  └───────────┘  │
│  └───────────┘  │         └─────────────────┘
└─────────────────┘
```

---

## 9. 开发路线图

### Phase 1: 核心基础 (MVP)
- [ ] 项目脚手架和基础架构
- [ ] API Server 和 SQLite 存储
- [ ] Pod 生命周期管理
- [ ] 基础容器运行时集成
- [ ] 简单的命令行工具

### Phase 2: 功能完善
- [ ] ReplicaSet 和 Deployment
- [ ] Service 和基础网络
- [ ] 调度器实现
- [ ] ConfigMap 和 Secret
- [ ] 健康检查和重启策略

### Phase 3: 多平台支持
- [ ] Windows 支持
- [ ] macOS 支持
- [ ] Android CLI 支持
- [ ] Android APK 原型

### Phase 4: 优化和扩展
- [ ] 性能优化
- [ ] 更多存储类型
- [ ] 监控和日志
- [ ] 文档完善

---

## 10. 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Android 容器限制 | 高 | 使用进程模拟或 rootless 模式 |
| 跨平台网络差异 | 中 | 抽象网络层，平台特定实现 |
| 资源限制下的稳定性 | 中 | 充分测试，优雅降级 |
| 与 K8s 生态兼容性 | 低 | 明确文档说明支持范围 |

---

## 11. 总结

MiniK8s 项目旨在创建一个轻量级、跨平台的容器编排工具，在保持与 Kubernetes 核心概念兼容的同时，大幅降低资源占用和部署复杂度。通过模块化设计和渐进式功能实现，项目既能满足学习教学需求，也可作为边缘计算等资源受限场景的轻量级解决方案。

关键成功因素：
1. 保持架构简洁，避免过度设计
2. 优先实现核心功能，确保稳定性
3. 充分测试跨平台兼容性
4. 完善的文档和示例

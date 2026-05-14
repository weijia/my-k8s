# MiniK8s

一个轻量级、跨平台的容器编排工具，与 Kubernetes 核心概念兼容。

## 特性

- **轻量级**: 单二进制文件，低资源占用
- **跨平台**: 支持 Linux、macOS、Windows
- **简单易用**: 零配置启动，使用 Docker 作为运行时
- **Kubernetes 兼容**: 熟悉的 API 和资源模型

## 快速开始

### 1. 安装 Docker

```bash
# Linux
curl -fsSL https://get.docker.com | sh

# macOS/Windows: 下载 Docker Desktop
```

### 2. 下载 MiniK8s

```bash
# 克隆仓库
git clone https://github.com/weijia/my-k8s.git
cd my-k8s

# 构建
make build
```

### 3. 启动 Server

```bash
./bin/minik8s server --bind :8080
```

### 4. 添加节点（可选）

```bash
./bin/minik8s agent --server http://localhost:8080 --name node-1
```

### 5. 创建 Pod

```bash
./bin/kubectl create -f examples/nginx-pod.yaml
```

### 6. 查看状态

```bash
./bin/kubectl get pods
./bin/kubectl get nodes
```

## 支持的命令

```bash
# Server 管理
minik8s server --bind :8080
minik8s agent --server http://localhost:8080

# 资源操作
kubectl create -f <file>
kubectl get pods
kubectl get nodes
kubectl delete pod <name>
kubectl logs <pod-name>
```

## 架构

```
┌─────────────────────────────────────┐
│         MiniK8s Control Plane        │
│  ┌─────────────┐  ┌─────────────┐   │
│  │  API Server │  │  Scheduler  │   │
│  │  (HTTP)     │  │  (简单)      │   │
│  └──────┬──────┘  └─────────────┘   │
│         │                            │
│         │ SQLite                     │
│         ▼                            │
│  ┌─────────────────────────────────┐ │
│  │      Docker Runtime             │ │
│  └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

## 开发

```bash
# 下载依赖
make deps

# 构建
make build

# 运行测试
make test

# 跨平台构建
make build-all
```

## 文档

- [架构设计](minik8s-architecture-design.md)
- [API 设计](docs/api-design.md)
- [数据模型](docs/data-model.md)
- [组件交互](docs/component-interaction.md)
- [存储和网络](docs/storage-network-design.md)
- [开发路线图](ROADMAP.md)
- [MVP 说明](MVP.md)

## 许可证

MIT License

# MiniK8s API 详细设计文档

## 1. API 概述

### 1.1 设计原则
- **RESTful 风格**: 遵循 REST 设计原则
- **Kubernetes 兼容**: API 路径和结构与 Kubernetes 保持一致
- **版本控制**: 通过 URL 路径进行版本控制
- **JSON 格式**: 请求和响应使用 JSON 格式

### 1.2 基础信息

| 属性 | 值 |
|------|-----|
| **基础 URL** | `http://localhost:8080/api/v1` |
| **内容类型** | `application/json` |
| **字符编码** | UTF-8 |

### 1.3 HTTP 方法映射

| HTTP 方法 | 操作 | 说明 |
|-----------|------|------|
| GET | 读取 | 获取资源或资源列表 |
| POST | 创建 | 创建新资源 |
| PUT | 更新 | 全量更新资源 |
| PATCH | 部分更新 | 部分更新资源（可选） |
| DELETE | 删除 | 删除资源 |

---

## 2. API 端点详细说明

### 2.1 Pod 管理

#### 2.1.1 创建 Pod
```http
POST /api/v1/namespaces/{namespace}/pods
```

**请求体**:
```json
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "nginx-pod",
    "namespace": "default",
    "labels": {
      "app": "nginx"
    }
  },
  "spec": {
    "containers": [
      {
        "name": "nginx",
        "image": "nginx:alpine",
        "ports": [
          {
            "containerPort": 80,
            "protocol": "TCP"
          }
        ],
        "resources": {
          "limits": {
            "cpu": "500m",
            "memory": "128Mi"
          },
          "requests": {
            "cpu": "100m",
            "memory": "64Mi"
          }
        },
        "env": [
          {
            "name": "NGINX_HOST",
            "value": "localhost"
          }
        ],
        "livenessProbe": {
          "httpGet": {
            "path": "/",
            "port": 80
          },
          "initialDelaySeconds": 30,
          "periodSeconds": 10
        },
        "readinessProbe": {
          "httpGet": {
            "path": "/",
            "port": 80
          },
          "initialDelaySeconds": 5,
          "periodSeconds": 5
        }
      }
    ],
    "restartPolicy": "Always"
  }
}
```

**响应** (201 Created):
```json
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "nginx-pod",
    "namespace": "default",
    "uid": "a3f5c8d2-1234-5678-9abc-def012345678",
    "creationTimestamp": "2026-05-14T10:30:00Z",
    "labels": {
      "app": "nginx"
    }
  },
  "spec": { ... },
  "status": {
    "phase": "Pending",
    "conditions": [],
    "containerStatuses": []
  }
}
```

#### 2.1.2 获取 Pod 列表
```http
GET /api/v1/namespaces/{namespace}/pods
```

**查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| labelSelector | string | 标签选择器，如 `app=nginx` |
| fieldSelector | string | 字段选择器（简化支持） |
| limit | integer | 返回结果数量限制 |
| continue | string | 分页令牌 |

**响应** (200 OK):
```json
{
  "apiVersion": "v1",
  "kind": "PodList",
  "metadata": {
    "resourceVersion": "12345"
  },
  "items": [
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": { ... },
      "spec": { ... },
      "status": { ... }
    }
  ]
}
```

#### 2.1.3 获取单个 Pod
```http
GET /api/v1/namespaces/{namespace}/pods/{name}
```

**响应** (200 OK): 单个 Pod 对象

#### 2.1.4 删除 Pod
```http
DELETE /api/v1/namespaces/{namespace}/pods/{name}
```

**查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| gracePeriodSeconds | integer | 优雅终止等待时间（秒） |

**响应** (200 OK):
```json
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": { ... }
}
```

#### 2.1.5 获取 Pod 日志
```http
GET /api/v1/namespaces/{namespace}/pods/{name}/log
```

**查询参数**:
| 参数 | 类型 | 说明 |
|------|------|------|
| container | string | 容器名称（多容器 Pod 必需） |
| follow | boolean | 是否持续跟踪 |
| tailLines | integer | 返回最后 N 行 |
| sinceSeconds | integer | 返回最近 N 秒的日志 |

**响应** (200 OK): 纯文本日志内容

---

### 2.2 Service 管理

#### 2.2.1 创建 Service
```http
POST /api/v1/namespaces/{namespace}/services
```

**请求体**:
```json
{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "nginx-service",
    "namespace": "default"
  },
  "spec": {
    "selector": {
      "app": "nginx"
    },
    "ports": [
      {
        "name": "http",
        "port": 80,
        "targetPort": 80,
        "protocol": "TCP"
      }
    ],
    "type": "ClusterIP",
    "sessionAffinity": "None"
  }
}
```

**Service 类型支持**:
| 类型 | 说明 |
|------|------|
| ClusterIP | 集群内部访问（默认） |
| NodePort | 通过节点端口暴露服务 |

#### 2.2.2 获取 Service 列表
```http
GET /api/v1/namespaces/{namespace}/services
```

#### 2.2.3 获取单个 Service
```http
GET /api/v1/namespaces/{namespace}/services/{name}
```

#### 2.2.4 删除 Service
```http
DELETE /api/v1/namespaces/{namespace}/services/{name}
```

---

### 2.3 Deployment 管理

#### 2.3.1 创建 Deployment
```http
POST /apis/apps/v1/namespaces/{namespace}/deployments
```

**请求体**:
```json
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "nginx-deployment",
    "namespace": "default",
    "labels": {
      "app": "nginx"
    }
  },
  "spec": {
    "replicas": 3,
    "selector": {
      "matchLabels": {
        "app": "nginx"
      }
    },
    "template": {
      "metadata": {
        "labels": {
          "app": "nginx"
        }
      },
      "spec": {
        "containers": [
          {
            "name": "nginx",
            "image": "nginx:alpine",
            "ports": [
              {
                "containerPort": 80
              }
            ],
            "resources": {
              "limits": {
                "cpu": "500m",
                "memory": "128Mi"
              }
            }
          }
        ]
      }
    },
    "strategy": {
      "type": "RollingUpdate",
      "rollingUpdate": {
        "maxSurge": "25%",
        "maxUnavailable": "25%"
      }
    }
  }
}
```

#### 2.3.2 扩缩容 Deployment
```http
PATCH /apis/apps/v1/namespaces/{namespace}/deployments/{name}/scale
```

**请求体**:
```json
{
  "spec": {
    "replicas": 5
  }
}
```

---

### 2.4 Node 管理

#### 2.4.1 获取节点列表
```http
GET /api/v1/nodes
```

**响应**:
```json
{
  "apiVersion": "v1",
  "kind": "NodeList",
  "items": [
    {
      "apiVersion": "v1",
      "kind": "Node",
      "metadata": {
        "name": "node-1",
        "labels": {
          "kubernetes.io/os": "linux"
        }
      },
      "status": {
        "capacity": {
          "cpu": "4",
          "memory": "8192Mi"
        },
        "allocatable": {
          "cpu": "3800m",
          "memory": "7500Mi"
        },
        "conditions": [
          {
            "type": "Ready",
            "status": "True"
          }
        ]
      }
    }
  ]
}
```

#### 2.4.2 获取单个节点
```http
GET /api/v1/nodes/{name}
```

---

### 2.5 ConfigMap 管理

#### 2.5.1 创建 ConfigMap
```http
POST /api/v1/namespaces/{namespace}/configmaps
```

**请求体**:
```json
{
  "apiVersion": "v1",
  "kind": "ConfigMap",
  "metadata": {
    "name": "app-config",
    "namespace": "default"
  },
  "data": {
    "database.properties": "host=localhost\nport=5432",
    "app.conf": "debug=true"
  }
}
```

#### 2.5.2 获取 ConfigMap 列表
```http
GET /api/v1/namespaces/{namespace}/configmaps
```

#### 2.5.3 获取单个 ConfigMap
```http
GET /api/v1/namespaces/{namespace}/configmaps/{name}
```

#### 2.5.4 删除 ConfigMap
```http
DELETE /api/v1/namespaces/{namespace}/configmaps/{name}
```

---

### 2.6 Secret 管理

#### 2.6.1 创建 Secret
```http
POST /api/v1/namespaces/{namespace}/secrets
```

**请求体**:
```json
{
  "apiVersion": "v1",
  "kind": "Secret",
  "metadata": {
    "name": "db-credentials",
    "namespace": "default"
  },
  "type": "Opaque",
  "data": {
    "username": "YWRtaW4=",
    "password": "cGFzc3dvcmQxMjM="
  }
}
```

**注意**: data 字段的值需要 Base64 编码。

---

### 2.7 Namespace 管理

#### 2.7.1 创建 Namespace
```http
POST /api/v1/namespaces
```

**请求体**:
```json
{
  "apiVersion": "v1",
  "kind": "Namespace",
  "metadata": {
    "name": "production"
  }
}
```

#### 2.7.2 获取 Namespace 列表
```http
GET /api/v1/namespaces
```

#### 2.7.3 删除 Namespace
```http
DELETE /api/v1/namespaces/{name}
```

---

### 2.8 Watch 机制（事件监听）

#### 2.8.1 监听 Pod 变化
```http
GET /api/v1/namespaces/{namespace}/pods?watch=true
```

**响应**: Server-Sent Events (SSE) 格式
```
data: {"type":"ADDED","object":{"apiVersion":"v1","kind":"Pod",...}}

data: {"type":"MODIFIED","object":{"apiVersion":"v1","kind":"Pod",...}}

data: {"type":"DELETED","object":{"apiVersion":"v1","kind":"Pod",...}}
```

**事件类型**:
| 类型 | 说明 |
|------|------|
| ADDED | 资源创建 |
| MODIFIED | 资源更新 |
| DELETED | 资源删除 |

---

## 3. 通用响应格式

### 3.1 成功响应

**列表响应**:
```json
{
  "apiVersion": "v1",
  "kind": "PodList",
  "metadata": {
    "resourceVersion": "12345",
    "selfLink": "/api/v1/namespaces/default/pods"
  },
  "items": [...]
}
```

**单个资源响应**:
```json
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": { ... },
  "spec": { ... },
  "status": { ... }
}
```

### 3.2 错误响应

**格式**:
```json
{
  "kind": "Status",
  "apiVersion": "v1",
  "metadata": {},
  "status": "Failure",
  "message": "pods \"nginx-pod\" not found",
  "reason": "NotFound",
  "details": {
    "name": "nginx-pod",
    "kind": "pods"
  },
  "code": 404
}
```

**HTTP 状态码**:
| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 204 | 成功（无返回内容） |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 409 | 资源冲突（如已存在） |
| 422 | 无法处理的实体 |
| 500 | 服务器内部错误 |

---

## 4. 字段说明

### 4.1 ObjectMeta（对象元数据）

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 资源名称（必需） |
| namespace | string | 命名空间（默认 default） |
| uid | string | 唯一标识符（系统生成） |
| resourceVersion | string | 资源版本（乐观锁） |
| creationTimestamp | string | 创建时间（ISO 8601） |
| labels | map[string]string | 标签 |
| annotations | map[string]string | 注解 |

### 4.2 PodSpec（Pod 规格）

| 字段 | 类型 | 说明 |
|------|------|------|
| containers | Container[] | 容器列表（必需） |
| restartPolicy | string | 重启策略：Always/OnFailure/Never |
| nodeSelector | map[string]string | 节点选择器 |
| volumes | Volume[] | 卷定义 |
| affinity | Affinity | 亲和性配置 |
| tolerations | Toleration[] | 容忍配置 |

### 4.3 Container（容器定义）

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 容器名称（必需） |
| image | string | 镜像地址（必需） |
| command | string[] | 启动命令 |
| args | string[] | 启动参数 |
| env | EnvVar[] | 环境变量 |
| ports | ContainerPort[] | 端口定义 |
| resources | ResourceRequirements | 资源限制 |
| volumeMounts | VolumeMount[] | 卷挂载 |
| livenessProbe | Probe | 存活探针 |
| readinessProbe | Probe | 就绪探针 |

### 4.4 ResourceRequirements（资源需求）

| 字段 | 类型 | 说明 |
|------|------|------|
| limits | ResourceList | 资源上限 |
| requests | ResourceList | 资源请求 |

**资源单位**:
- CPU: `m`（毫核），如 `500m` = 0.5 核
- 内存: `Mi`, `Gi`，如 `128Mi`, `1Gi`

---

## 5. 健康检查端点

### 5.1 健康状态
```http
GET /healthz
```

**响应**: `200 OK` - "ok"

### 5.2 就绪状态
```http
GET /readyz
```

**响应**: `200 OK` - "ok"

---

## 6. API 版本说明

| API 组 | 版本 | 资源 |
|--------|------|------|
| core | v1 | Pod, Service, ConfigMap, Secret, Node, Namespace |
| apps | v1 | Deployment, ReplicaSet |

---

## 7. 与 Kubernetes API 的差异

| 特性 | Kubernetes | MiniK8s |
|------|------------|---------|
| 认证 | 多种方式 | 可选简单 Token 或无认证 |
| 授权 | RBAC | 无 |
| Admission | Webhook | 无 |
| CRD | 支持 | 不支持 |
| 子资源 | 完整 | 简化 |
| 字段验证 | 严格 | 宽松 |
| 分页 | 完整 | 基础支持 |

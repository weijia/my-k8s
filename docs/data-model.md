# MiniK8s 数据模型设计文档

## 1. 数据模型概述

### 1.1 设计原则
- **与 Kubernetes 兼容**: 使用相同的核心字段和结构
- **简化实现**: 去除不必要的复杂字段
- **SQLite 友好**: 支持序列化为 JSON 存储
- **类型安全**: 完整的 Go struct 定义

### 1.2 存储方式
- **主存储**: SQLite（单文件，零配置）
- **内存缓存**: 运行时对象缓存
- **序列化格式**: JSON

---

## 2. 核心数据模型

### 2.1 基础类型

#### 2.1.1 TypeMeta（类型元数据）
```go
type TypeMeta struct {
    APIVersion string `json:"apiVersion,omitempty"`
    Kind       string `json:"kind,omitempty"`
}
```

#### 2.1.2 ObjectMeta（对象元数据）
```go
type ObjectMeta struct {
    Name              string            `json:"name,omitempty"`
    Namespace         string            `json:"namespace,omitempty"`
    UID               string            `json:"uid,omitempty"`
    ResourceVersion   string            `json:"resourceVersion,omitempty"`
    CreationTimestamp string            `json:"creationTimestamp,omitempty"`
    Labels            map[string]string `json:"labels,omitempty"`
    Annotations       map[string]string `json:"annotations,omitempty"`
}
```

#### 2.1.3 ObjectReference（对象引用）
```go
type ObjectReference struct {
    Kind      string `json:"kind,omitempty"`
    Name      string `json:"name,omitempty"`
    Namespace string `json:"namespace,omitempty"`
    UID       string `json:"uid,omitempty"`
}
```

---

### 2.2 Pod 数据模型

#### 2.2.1 Pod（完整定义）
```go
type Pod struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Spec       PodSpec   `json:"spec,omitempty"`
    Status     PodStatus `json:"status,omitempty"`
}

// PodList Pod 列表
type PodList struct {
    TypeMeta `json:",inline"`
    ListMeta `json:"metadata,omitempty"`
    Items    []Pod `json:"items"`
}

// ListMeta 列表元数据
type ListMeta struct {
    ResourceVersion string `json:"resourceVersion,omitempty"`
    Continue        string `json:"continue,omitempty"`
}
```

#### 2.2.2 PodSpec（Pod 规格）
```go
type PodSpec struct {
    Containers       []Container       `json:"containers"`
    RestartPolicy    RestartPolicy     `json:"restartPolicy,omitempty"`
    NodeSelector     map[string]string `json:"nodeSelector,omitempty"`
    NodeName         string            `json:"nodeName,omitempty"`
    Volumes          []Volume          `json:"volumes,omitempty"`
    Affinity         *Affinity         `json:"affinity,omitempty"`
    Tolerations      []Toleration      `json:"tolerations,omitempty"`
}

// RestartPolicy 重启策略
type RestartPolicy string

const (
    RestartPolicyAlways    RestartPolicy = "Always"
    RestartPolicyOnFailure RestartPolicy = "OnFailure"
    RestartPolicyNever     RestartPolicy = "Never"
)
```

#### 2.2.3 Container（容器定义）
```go
type Container struct {
    Name           string             `json:"name"`
    Image          string             `json:"image"`
    Command        []string           `json:"command,omitempty"`
    Args           []string           `json:"args,omitempty"`
    WorkingDir     string             `json:"workingDir,omitempty"`
    Ports          []ContainerPort    `json:"ports,omitempty"`
    Env            []EnvVar           `json:"env,omitempty"`
    Resources      ResourceRequirements `json:"resources,omitempty"`
    VolumeMounts   []VolumeMount      `json:"volumeMounts,omitempty"`
    LivenessProbe  *Probe             `json:"livenessProbe,omitempty"`
    ReadinessProbe *Probe             `json:"readinessProbe,omitempty"`
}

// ContainerPort 容器端口
type ContainerPort struct {
    Name          string `json:"name,omitempty"`
    ContainerPort int32  `json:"containerPort"`
    Protocol      string `json:"protocol,omitempty"`
}

// EnvVar 环境变量
type EnvVar struct {
    Name  string `json:"name"`
    Value string `json:"value,omitempty"`
}

// ResourceRequirements 资源需求
type ResourceRequirements struct {
    Limits   ResourceList `json:"limits,omitempty"`
    Requests ResourceList `json:"requests,omitempty"`
}

// ResourceList 资源列表
type ResourceList map[string]string

// VolumeMount 卷挂载
type VolumeMount struct {
    Name      string `json:"name"`
    MountPath string `json:"mountPath"`
    ReadOnly  bool   `json:"readOnly,omitempty"`
}

// Probe 健康探针
type Probe struct {
    HTTPGet             *HTTPGetAction `json:"httpGet,omitempty"`
    TCPSocket           *TCPSocketAction `json:"tcpSocket,omitempty"`
    InitialDelaySeconds int32          `json:"initialDelaySeconds,omitempty"`
    PeriodSeconds       int32          `json:"periodSeconds,omitempty"`
    TimeoutSeconds      int32          `json:"timeoutSeconds,omitempty"`
    FailureThreshold    int32          `json:"failureThreshold,omitempty"`
}

// HTTPGetAction HTTP 探测
type HTTPGetAction struct {
    Path   string `json:"path,omitempty"`
    Port   int32  `json:"port"`
    Host   string `json:"host,omitempty"`
    Scheme string `json:"scheme,omitempty"`
}

// TCPSocketAction TCP 探测
type TCPSocketAction struct {
    Port int32 `json:"port"`
}
```

#### 2.2.4 PodStatus（Pod 状态）
```go
type PodStatus struct {
    Phase             PodPhase          `json:"phase,omitempty"`
    Conditions        []PodCondition    `json:"conditions,omitempty"`
    Message           string            `json:"message,omitempty"`
    Reason            string            `json:"reason,omitempty"`
    HostIP            string            `json:"hostIP,omitempty"`
    PodIP             string            `json:"podIP,omitempty"`
    StartTime         string            `json:"startTime,omitempty"`
    ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
}

// PodPhase Pod 阶段
type PodPhase string

const (
    PodPending   PodPhase = "Pending"
    PodRunning   PodPhase = "Running"
    PodSucceeded PodPhase = "Succeeded"
    PodFailed    PodPhase = "Failed"
    PodUnknown   PodPhase = "Unknown"
)

// PodCondition Pod 条件
type PodCondition struct {
    Type   PodConditionType `json:"type"`
    Status ConditionStatus  `json:"status"`
}

// PodConditionType 条件类型
type PodConditionType string

const (
    PodScheduled   PodConditionType = "PodScheduled"
    PodInitialized PodConditionType = "Initialized"
    ContainersReady PodConditionType = "ContainersReady"
    PodReady       PodConditionType = "Ready"
)

// ConditionStatus 条件状态
type ConditionStatus string

const (
    ConditionTrue    ConditionStatus = "True"
    ConditionFalse   ConditionStatus = "False"
    ConditionUnknown ConditionStatus = "Unknown"
)

// ContainerStatus 容器状态
type ContainerStatus struct {
    Name        string         `json:"name"`
    State       ContainerState `json:"state,omitempty"`
    Ready       bool           `json:"ready"`
    RestartCount int32         `json:"restartCount"`
    Image       string         `json:"image"`
}

// ContainerState 容器状态
type ContainerState struct {
    Waiting    *ContainerStateWaiting    `json:"waiting,omitempty"`
    Running    *ContainerStateRunning    `json:"running,omitempty"`
    Terminated *ContainerStateTerminated `json:"terminated,omitempty"`
}

type ContainerStateWaiting struct {
    Reason  string `json:"reason,omitempty"`
    Message string `json:"message,omitempty"`
}

type ContainerStateRunning struct {
    StartedAt string `json:"startedAt,omitempty"`
}

type ContainerStateTerminated struct {
    ExitCode   int32  `json:"exitCode"`
    Signal     int32  `json:"signal,omitempty"`
    Reason     string `json:"reason,omitempty"`
    Message    string `json:"message,omitempty"`
    StartedAt  string `json:"startedAt,omitempty"`
    FinishedAt string `json:"finishedAt,omitempty"`
}
```

---

### 2.3 Service 数据模型

#### 2.3.1 Service（完整定义）
```go
type Service struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Spec       ServiceSpec   `json:"spec,omitempty"`
    Status     ServiceStatus `json:"status,omitempty"`
}

type ServiceList struct {
    TypeMeta `json:",inline"`
    ListMeta `json:"metadata,omitempty"`
    Items    []Service `json:"items"`
}
```

#### 2.3.2 ServiceSpec（Service 规格）
```go
type ServiceSpec struct {
    Type           ServiceType       `json:"type,omitempty"`
    Selector       map[string]string `json:"selector,omitempty"`
    Ports          []ServicePort     `json:"ports,omitempty"`
    ClusterIP      string            `json:"clusterIP,omitempty"`
    SessionAffinity string           `json:"sessionAffinity,omitempty"`
}

// ServiceType 服务类型
type ServiceType string

const (
    ServiceTypeClusterIP ServiceType = "ClusterIP"
    ServiceTypeNodePort  ServiceType = "NodePort"
)

// ServicePort 服务端口
type ServicePort struct {
    Name       string `json:"name,omitempty"`
    Protocol   string `json:"protocol,omitempty"`
    Port       int32  `json:"port"`
    TargetPort int32  `json:"targetPort,omitempty"`
    NodePort   int32  `json:"nodePort,omitempty"`
}
```

#### 2.3.3 ServiceStatus（Service 状态）
```go
type ServiceStatus struct {
    LoadBalancer LoadBalancerStatus `json:"loadBalancer,omitempty"`
}

type LoadBalancerStatus struct {
    Ingress []LoadBalancerIngress `json:"ingress,omitempty"`
}

type LoadBalancerIngress struct {
    IP       string `json:"ip,omitempty"`
    Hostname string `json:"hostname,omitempty"`
}
```

#### 2.3.4 Endpoints（端点）
```go
type Endpoints struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Subsets    []EndpointSubset `json:"subsets"`
}

type EndpointSubset struct {
    Addresses []EndpointAddress `json:"addresses,omitempty"`
    Ports     []EndpointPort    `json:"ports,omitempty"`
}

type EndpointAddress struct {
    IP        string `json:"ip"`
    NodeName  string `json:"nodeName,omitempty"`
    TargetRef *ObjectReference `json:"targetRef,omitempty"`
}

type EndpointPort struct {
    Name     string `json:"name,omitempty"`
    Port     int32  `json:"port"`
    Protocol string `json:"protocol,omitempty"`
}
```

---

### 2.4 Deployment 数据模型

#### 2.4.1 Deployment（完整定义）
```go
type Deployment struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Spec       DeploymentSpec   `json:"spec,omitempty"`
    Status     DeploymentStatus `json:"status,omitempty"`
}

type DeploymentList struct {
    TypeMeta `json:",inline"`
    ListMeta `json:"metadata,omitempty"`
    Items    []Deployment `json:"items"`
}
```

#### 2.4.2 DeploymentSpec（Deployment 规格）
```go
type DeploymentSpec struct {
    Replicas        int32                `json:"replicas,omitempty"`
    Selector        *LabelSelector       `json:"selector"`
    Template        PodTemplateSpec      `json:"template"`
    Strategy        DeploymentStrategy   `json:"strategy,omitempty"`
}

// LabelSelector 标签选择器
type LabelSelector struct {
    MatchLabels      map[string]string    `json:"matchLabels,omitempty"`
    MatchExpressions []LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// LabelSelectorRequirement 选择器要求
type LabelSelectorRequirement struct {
    Key      string   `json:"key"`
    Operator string   `json:"operator"`
    Values   []string `json:"values,omitempty"`
}

// PodTemplateSpec Pod 模板规格
type PodTemplateSpec struct {
    ObjectMeta `json:"metadata,omitempty"`
    Spec       PodSpec `json:"spec,omitempty"`
}

// DeploymentStrategy 部署策略
type DeploymentStrategy struct {
    Type          DeploymentStrategyType `json:"type,omitempty"`
    RollingUpdate *RollingUpdateDeployment `json:"rollingUpdate,omitempty"`
}

// DeploymentStrategyType 策略类型
type DeploymentStrategyType string

const (
    RecreateDeploymentStrategyType  DeploymentStrategyType = "Recreate"
    RollingUpdateDeploymentStrategyType DeploymentStrategyType = "RollingUpdate"
)

// RollingUpdateDeployment 滚动更新配置
type RollingUpdateDeployment struct {
    MaxUnavailable *IntOrString `json:"maxUnavailable,omitempty"`
    MaxSurge       *IntOrString `json:"maxSurge,omitempty"`
}

// IntOrString 可以是整数或字符串
type IntOrString struct {
    Type   Type   `json:"type,omitempty"`
    IntVal int32  `json:"intVal,omitempty"`
    StrVal string `json:"strVal,omitempty"`
}
```

#### 2.4.3 DeploymentStatus（Deployment 状态）
```go
type DeploymentStatus struct {
    ObservedGeneration  int64 `json:"observedGeneration,omitempty"`
    Replicas            int32 `json:"replicas,omitempty"`
    UpdatedReplicas     int32 `json:"updatedReplicas,omitempty"`
    ReadyReplicas       int32 `json:"readyReplicas,omitempty"`
    AvailableReplicas   int32 `json:"availableReplicas,omitempty"`
    UnavailableReplicas int32 `json:"unavailableReplicas,omitempty"`
}
```

#### 2.4.4 ReplicaSet（副本集）
```go
type ReplicaSet struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Spec       ReplicaSetSpec   `json:"spec,omitempty"`
    Status     ReplicaSetStatus `json:"status,omitempty"`
}

type ReplicaSetSpec struct {
    Replicas int32          `json:"replicas,omitempty"`
    Selector *LabelSelector `json:"selector"`
    Template PodTemplateSpec `json:"template,omitempty"`
}

type ReplicaSetStatus struct {
    Replicas             int32 `json:"replicas,omitempty"`
    FullyLabeledReplicas int32 `json:"fullyLabeledReplicas,omitempty"`
    ReadyReplicas        int32 `json:"readyReplicas,omitempty"`
    AvailableReplicas    int32 `json:"availableReplicas,omitempty"`
}
```

---

### 2.5 Node 数据模型

#### 2.5.1 Node（完整定义）
```go
type Node struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Spec       NodeSpec   `json:"spec,omitempty"`
    Status     NodeStatus `json:"status,omitempty"`
}

type NodeList struct {
    TypeMeta `json:",inline"`
    ListMeta `json:"metadata,omitempty"`
    Items    []Node `json:"items"`
}
```

#### 2.5.2 NodeSpec（Node 规格）
```go
type NodeSpec struct {
    Unschedulable bool `json:"unschedulable,omitempty"`
}
```

#### 2.5.3 NodeStatus（Node 状态）
```go
type NodeStatus struct {
    Capacity    ResourceList   `json:"capacity,omitempty"`
    Allocatable ResourceList   `json:"allocatable,omitempty"`
    Conditions  []NodeCondition `json:"conditions,omitempty"`
    Addresses   []NodeAddress  `json:"addresses,omitempty"`
    NodeInfo    NodeSystemInfo `json:"nodeInfo,omitempty"`
}

// NodeCondition 节点条件
type NodeCondition struct {
    Type   NodeConditionType `json:"type"`
    Status ConditionStatus   `json:"status"`
    Reason string            `json:"reason,omitempty"`
}

// NodeConditionType 节点条件类型
type NodeConditionType string

const (
    NodeReady              NodeConditionType = "Ready"
    NodeMemoryPressure     NodeConditionType = "MemoryPressure"
    NodeDiskPressure       NodeConditionType = "DiskPressure"
)

// NodeAddress 节点地址
type NodeAddress struct {
    Type    NodeAddressType `json:"type"`
    Address string          `json:"address"`
}

// NodeAddressType 地址类型
type NodeAddressType string

const (
    NodeHostName    NodeAddressType = "Hostname"
    NodeExternalIP  NodeAddressType = "ExternalIP"
    NodeInternalIP  NodeAddressType = "InternalIP"
)

// NodeSystemInfo 节点系统信息
type NodeSystemInfo struct {
    MachineID               string `json:"machineID,omitempty"`
    SystemUUID              string `json:"systemUUID,omitempty"`
    BootID                  string `json:"bootID,omitempty"`
    KernelVersion           string `json:"kernelVersion,omitempty"`
    OSImage                 string `json:"osImage,omitempty"`
    ContainerRuntimeVersion string `json:"containerRuntimeVersion,omitempty"`
    KubeletVersion          string `json:"kubeletVersion,omitempty"`
    OperatingSystem         string `json:"operatingSystem,omitempty"`
    Architecture            string `json:"architecture,omitempty"`
}
```

---

### 2.6 ConfigMap 和 Secret 数据模型

#### 2.6.1 ConfigMap
```go
type ConfigMap struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Data       map[string]string `json:"data,omitempty"`
}

type ConfigMapList struct {
    TypeMeta `json:",inline"`
    ListMeta `json:"metadata,omitempty"`
    Items    []ConfigMap `json:"items"`
}
```

#### 2.6.2 Secret
```go
type Secret struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Type       SecretType        `json:"type,omitempty"`
    Data       map[string][]byte `json:"data,omitempty"`
}

// SecretType Secret 类型
type SecretType string

const (
    SecretTypeOpaque SecretType = "Opaque"
)

type SecretList struct {
    TypeMeta `json:",inline"`
    ListMeta `json:"metadata,omitempty"`
    Items    []Secret `json:"items"`
}
```

---

### 2.7 Namespace 数据模型

```go
type Namespace struct {
    TypeMeta   `json:",inline"`
    ObjectMeta `json:"metadata,omitempty"`
    Spec       NamespaceSpec   `json:"spec,omitempty"`
    Status     NamespaceStatus `json:"status,omitempty"`
}

type NamespaceSpec struct {
    Finalizers []string `json:"finalizers,omitempty"`
}

type NamespaceStatus struct {
    Phase NamespacePhase `json:"phase,omitempty"`
}

// NamespacePhase 命名空间阶段
type NamespacePhase string

const (
    NamespaceActive   NamespacePhase = "Active"
    NamespaceTerminating NamespacePhase = "Terminating"
)

type NamespaceList struct {
    TypeMeta `json:",inline"`
    ListMeta `json:"metadata,omitempty"`
    Items    []Namespace `json:"items"`
}
```

---

### 2.8 Volume 数据模型

```go
// Volume 卷定义
type Volume struct {
    Name                  string                `json:"name"`
    EmptyDir             *EmptyDirVolumeSource `json:"emptyDir,omitempty"`
    HostPath             *HostPathVolumeSource `json:"hostPath,omitempty"`
    ConfigMap            *ConfigMapVolumeSource `json:"configMap,omitempty"`
    Secret               *SecretVolumeSource    `json:"secret,omitempty"`
}

// EmptyDirVolumeSource emptyDir 卷
type EmptyDirVolumeSource struct {
    Medium string `json:"medium,omitempty"`
}

// HostPathVolumeSource hostPath 卷
type HostPathVolumeSource struct {
    Path string `json:"path"`
    Type string `json:"type,omitempty"`
}

// ConfigMapVolumeSource ConfigMap 卷
type ConfigMapVolumeSource struct {
    Name string `json:"name"`
}

// SecretVolumeSource Secret 卷
type SecretVolumeSource struct {
    SecretName string `json:"secretName"`
}
```

---

### 2.9 调度相关数据模型

```go
// Affinity 亲和性
type Affinity struct {
    NodeAffinity    *NodeAffinity    `json:"nodeAffinity,omitempty"`
    PodAffinity     *PodAffinity     `json:"podAffinity,omitempty"`
    PodAntiAffinity *PodAntiAffinity `json:"podAntiAffinity,omitempty"`
}

// NodeAffinity 节点亲和性
type NodeAffinity struct {
    RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelector 节点选择器
type NodeSelector struct {
    NodeSelectorTerms []NodeSelectorTerm `json:"nodeSelectorTerms"`
}

// NodeSelectorTerm 节点选择条件
type NodeSelectorTerm struct {
    MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty"`
    MatchFields      []NodeSelectorRequirement `json:"matchFields,omitempty"`
}

// NodeSelectorRequirement 节点选择要求
type NodeSelectorRequirement struct {
    Key      string   `json:"key"`
    Operator string   `json:"operator"`
    Values   []string `json:"values,omitempty"`
}

// PodAffinity Pod 亲和性（简化）
type PodAffinity struct {
    RequiredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAntiAffinity Pod 反亲和性（简化）
type PodAntiAffinity struct {
    RequiredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinityTerm Pod 亲和性条件
type PodAffinityTerm struct {
    LabelSelector *LabelSelector `json:"labelSelector,omitempty"`
    Namespaces    []string       `json:"namespaces,omitempty"`
    TopologyKey   string         `json:"topologyKey"`
}

// Toleration 容忍
type Toleration struct {
    Key               string `json:"key,omitempty"`
    Operator          string `json:"operator,omitempty"`
    Value             string `json:"value,omitempty"`
    Effect            string `json:"effect,omitempty"`
    TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}
```

---

## 3. 数据库 Schema 设计

### 3.1 SQLite 表结构

```sql
-- 资源表（通用存储）
CREATE TABLE resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_version TEXT NOT NULL,
    kind TEXT NOT NULL,
    namespace TEXT,
    name TEXT NOT NULL,
    uid TEXT UNIQUE NOT NULL,
    resource_version TEXT,
    creation_timestamp TEXT,
    data JSON NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace, name, kind)
);

-- 索引
CREATE INDEX idx_resources_kind ON resources(kind);
CREATE INDEX idx_resources_namespace ON resources(namespace);
CREATE INDEX idx_resources_uid ON resources(uid);

-- 事件表
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uid TEXT NOT NULL,
    type TEXT NOT NULL,  -- Normal, Warning
    reason TEXT,
    message TEXT,
    involved_object_kind TEXT,
    involved_object_name TEXT,
    involved_object_namespace TEXT,
    count INTEGER DEFAULT 1,
    first_timestamp TEXT,
    last_timestamp TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_events_object ON events(involved_object_kind, involved_object_name, involved_object_namespace);
```

### 3.2 存储接口设计

```go
// Storage 存储接口
type Storage interface {
    // Create 创建资源
    Create(obj Object) error
    
    // Get 获取资源
    Get(kind, namespace, name string) (Object, error)
    
    // Update 更新资源
    Update(obj Object) error
    
    // Delete 删除资源
    Delete(kind, namespace, name string) error
    
    // List 列出资源
    List(kind, namespace string, selector LabelSelector) ([]Object, error)
    
    // Watch 监听资源变化
    Watch(kind, namespace string) (WatchInterface, error)
    
    // Close 关闭存储
    Close() error
}

// Object 资源对象接口
type Object interface {
    GetUID() string
    GetName() string
    GetNamespace() string
    GetResourceVersion() string
    SetResourceVersion(string)
    GetLabels() map[string]string
}

// WatchInterface 监听接口
type WatchInterface interface {
    ResultChan() <-chan Event
    Stop()
}

// Event 事件
type Event struct {
    Type   EventType
    Object Object
}

// EventType 事件类型
type EventType string

const (
    Added    EventType = "ADDED"
    Modified EventType = "MODIFIED"
    Deleted  EventType = "DELETED"
)
```

---

## 4. 与 Kubernetes 的差异

| 特性 | Kubernetes | MiniK8s |
|------|------------|---------|
| 存储后端 | etcd | SQLite |
| 资源版本 | 64位整数 | 字符串/UUID |
| UID 生成 | 标准 UUID | 简化 UUID |
| 字段验证 | 严格 | 宽松 |
| 默认值 | 完整 | 简化 |
| 注解支持 | 完整 | 基础 |
| Finalizers | 支持 | 简化 |
| OwnerReferences | 完整 | 简化 |

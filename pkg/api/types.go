package api

import "time"

// TypeMeta 类型元数据
type TypeMeta struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion"`
	Kind       string `json:"kind,omitempty" yaml:"kind"`
}

// ObjectMeta 对象元数据
type ObjectMeta struct {
	Name              string            `json:"name,omitempty" yaml:"name"`
	Namespace         string            `json:"namespace,omitempty" yaml:"namespace"`
	UID               string            `json:"uid,omitempty" yaml:"uid"`
	ResourceVersion   string            `json:"resourceVersion,omitempty" yaml:"resourceVersion"`
	CreationTimestamp time.Time         `json:"creationTimestamp,omitempty" yaml:"creationTimestamp"`
	Labels            map[string]string `json:"labels,omitempty" yaml:"labels"`
}

// Pod 定义
type Pod struct {
	TypeMeta   `json:",inline" yaml:",inline"`
	ObjectMeta `json:"metadata,omitempty" yaml:"metadata"`
	Spec       PodSpec   `json:"spec,omitempty" yaml:"spec"`
	Status     PodStatus `json:"status,omitempty" yaml:"status"`
}

// PodSpec Pod 规格
type PodSpec struct {
	Containers    []Container `json:"containers" yaml:"containers"`
	NodeName      string      `json:"nodeName,omitempty" yaml:"nodeName"`
	RestartPolicy string      `json:"restartPolicy,omitempty" yaml:"restartPolicy"`
}

// Container 容器定义
type Container struct {
	Name    string   `json:"name" yaml:"name"`
	Image   string   `json:"image" yaml:"image"`
	Command []string `json:"command,omitempty" yaml:"command"`
	Args    []string `json:"args,omitempty" yaml:"args"`
	Ports   []Port   `json:"ports,omitempty" yaml:"ports"`
	Env     []EnvVar `json:"env,omitempty" yaml:"env"`
}

// Port 端口定义
type Port struct {
	ContainerPort int32  `json:"containerPort" yaml:"containerPort"`
	HostPort      int32  `json:"hostPort,omitempty" yaml:"hostPort"`
	Protocol      string `json:"protocol,omitempty" yaml:"protocol"`
}

// EnvVar 环境变量
type EnvVar struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

// PodStatus Pod 状态
type PodStatus struct {
	Phase      string            `json:"phase,omitempty" yaml:"phase"`
	PodIP      string            `json:"podIP,omitempty" yaml:"podIP"`
	HostIP     string            `json:"hostIP,omitempty" yaml:"hostIP"`
	Conditions []PodCondition    `json:"conditions,omitempty" yaml:"conditions"`
	Containers []ContainerStatus `json:"containerStatuses,omitempty" yaml:"containerStatuses"`
}

// PodCondition Pod 条件
type PodCondition struct {
	Type   string `json:"type" yaml:"type"`
	Status string `json:"status" yaml:"status"`
}

// ContainerStatus 容器状态
type ContainerStatus struct {
	Name        string `json:"name" yaml:"name"`
	State       string `json:"state" yaml:"state"`
	ContainerID string `json:"containerID,omitempty" yaml:"containerID"`
	Image       string `json:"image" yaml:"image"`
}

// PodPhase 定义
const (
	PodPending   = "Pending"
	PodRunning   = "Running"
	PodSucceeded = "Succeeded"
	PodFailed    = "Failed"
	PodUnknown   = "Unknown"
)

// Node 定义
type Node struct {
	TypeMeta   `json:",inline" yaml:",inline"`
	ObjectMeta `json:"metadata,omitempty" yaml:"metadata"`
	Spec       NodeSpec   `json:"spec,omitempty" yaml:"spec"`
	Status     NodeStatus `json:"status,omitempty" yaml:"status"`
}

// NodeSpec Node 规格
type NodeSpec struct {
	Unschedulable bool `json:"unschedulable,omitempty" yaml:"unschedulable"`
}

// NodeStatus Node 状态
type NodeStatus struct {
	Capacity    map[string]string `json:"capacity,omitempty" yaml:"capacity"`
	Allocatable map[string]string `json:"allocatable,omitempty" yaml:"allocatable"`
	Conditions  []NodeCondition   `json:"conditions,omitempty" yaml:"conditions"`
	Addresses   []NodeAddress     `json:"addresses,omitempty" yaml:"addresses"`
}

// NodeCondition Node 条件
type NodeCondition struct {
	Type   string `json:"type" yaml:"type"`
	Status string `json:"status" yaml:"status"`
}

// NodeAddress Node 地址
type NodeAddress struct {
	Type    string `json:"type" yaml:"type"`
	Address string `json:"address" yaml:"address"`
}

// Node 条件类型
const (
	NodeReady = "Ready"
)

// PodList Pod 列表
type PodList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Items    []Pod `json:"items" yaml:"items"`
}

// NodeList Node 列表
type NodeList struct {
	TypeMeta `json:",inline" yaml:",inline"`
	Items    []Node `json:"items" yaml:"items"`
}

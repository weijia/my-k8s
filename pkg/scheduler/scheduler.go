package scheduler

import (
	"fmt"
	"math/rand/v2"

	"github.com/weijia/my-k8s/pkg/api"
	"github.com/weijia/my-k8s/pkg/storage"
)

// Scheduler 调度器
type Scheduler struct {
	store *storage.Store
}

// NewScheduler 创建调度器
func NewScheduler(store *storage.Store) *Scheduler {
	return &Scheduler{store: store}
}

// Schedule 为 Pod 选择节点
func (s *Scheduler) Schedule(pod *api.Pod) (string, error) {
	// 如果 Pod 已指定节点，直接使用
	if pod.Spec.NodeName != "" {
		return pod.Spec.NodeName, nil
	}

	// 获取所有可用节点
	nodes, err := s.store.ListNodes()
	if err != nil {
		return "", err
	}

	// 过滤可用节点
	availableNodes := s.filterNodes(pod, nodes)
	if len(availableNodes) == 0 {
		// MVP 简化：如果没有可用节点，使用 localhost
		return "localhost", nil
	}

	// 选择节点（MVP 使用随机选择）
	selected := s.selectNode(availableNodes)
	return selected.Name, nil
}

// filterNodes 过滤可用节点
func (s *Scheduler) filterNodes(pod *api.Pod, nodes []api.Node) []api.Node {
	var available []api.Node
	for _, node := range nodes {
		// 检查节点是否可调度
		if node.Spec.Unschedulable {
			continue
		}

		// 检查节点是否就绪
		ready := false
		for _, cond := range node.Status.Conditions {
			if cond.Type == api.NodeReady && cond.Status == "True" {
				ready = true
				break
			}
		}
		if !ready {
			continue
		}

		available = append(available, node)
	}
	return available
}

// selectNode 选择节点（随机）
func (s *Scheduler) selectNode(nodes []api.Node) api.Node {
	return nodes[rand.IntN(len(nodes))]
}

// GetNodeStatus 获取节点状态
func (s *Scheduler) GetNodeStatus(nodeName string) (*api.NodeStatus, error) {
	node, err := s.store.GetNode(nodeName)
	if err != nil {
		return nil, fmt.Errorf("node not found: %s", nodeName)
	}
	return &node.Status, nil
}

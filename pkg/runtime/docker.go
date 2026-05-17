package runtime

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/weijia/my-k8s/pkg/api"
)

// DockerRuntime Docker 运行时
type DockerRuntime struct {
	client *client.Client
}

// NewDockerRuntime 创建 Docker 运行时
func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRuntime{client: cli}, nil
}

// CreatePod 创建 Pod（创建所有容器）
func (d *DockerRuntime) CreatePod(pod *api.Pod) error {
	ctx := context.Background()

	// 创建网络（简化：使用 host 网络）
	for i, c := range pod.Spec.Containers {
		containerName := fmt.Sprintf("%s_%s_%s", pod.Namespace, pod.Name, c.Name)

		// 准备端口映射
		portBindings := make(nat.PortMap)
		for _, port := range c.Ports {
			if port.HostPort > 0 {
				proto := "tcp"
				if port.Protocol == "udp" {
					proto = "udp"
				}
				containerPort, _ := nat.NewPort(proto, fmt.Sprintf("%d", port.ContainerPort))
				portBindings[containerPort] = []nat.PortBinding{
					{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", port.HostPort)},
				}
			}
		}

		// 准备环境变量
		env := make([]string, 0, len(c.Env))
		for _, e := range c.Env {
			env = append(env, fmt.Sprintf("%s=%s", e.Name, e.Value))
		}

		// 准备命令
		cmd := c.Command
		if len(c.Args) > 0 {
			cmd = append(cmd, c.Args...)
		}

		// 创建容器配置
		config := &container.Config{
			Image: c.Image,
			Cmd:   cmd,
			Env:   env,
			Labels: map[string]string{
				"minik8s.io/pod":       pod.Name,
				"minik8s.io/namespace": pod.Namespace,
				"minik8s.io/container": c.Name,
			},
		}

		hostConfig := &container.HostConfig{
			NetworkMode:  container.NetworkMode("host"),
			PortBindings: portBindings,
		}

		// 拉取镜像
		if err := d.pullImage(ctx, c.Image); err != nil {
			return fmt.Errorf("failed to pull image %s: %v", c.Image, err)
		}

		// 创建容器
		resp, err := d.client.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
		if err != nil {
			return fmt.Errorf("failed to create container: %v", err)
		}

		// 启动容器
		if err := d.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			return fmt.Errorf("failed to start container: %v", err)
		}

		// 更新状态
		pod.Status.Containers = append(pod.Status.Containers, api.ContainerStatus{
			Name:        c.Name,
			State:       "running",
			ContainerID: resp.ID,
			Image:       c.Image,
		})

		if i == 0 {
			pod.Status.Phase = api.PodRunning
		}
	}

	return nil
}

// DeletePod 删除 Pod（删除所有容器）
func (d *DockerRuntime) DeletePod(pod *api.Pod) error {
	ctx := context.Background()

	for _, c := range pod.Spec.Containers {
		containerName := fmt.Sprintf("%s_%s_%s", pod.Namespace, pod.Name, c.Name)

		// 查找容器
		filter := filters.NewArgs()
		filter.Add("name", containerName)
		containers, err := d.client.ContainerList(ctx, types.ContainerListOptions{
			All:     true,
			Filters: filter,
		})
		if err != nil {
			continue
		}

		for _, cont := range containers {
			// 停止容器
			timeout := 10
			d.client.ContainerStop(ctx, cont.ID, container.StopOptions{Timeout: &timeout})

			// 删除容器
			d.client.ContainerRemove(ctx, cont.ID, types.ContainerRemoveOptions{
				Force: true,
			})
		}
	}

	return nil
}

// GetPodStatus 获取 Pod 状态
func (d *DockerRuntime) GetPodStatus(pod *api.Pod) (*api.PodStatus, error) {
	ctx := context.Background()
	status := &api.PodStatus{
		Phase:      api.PodPending,
		Containers: make([]api.ContainerStatus, 0, len(pod.Spec.Containers)),
	}

	allRunning := true
	for _, c := range pod.Spec.Containers {
		containerName := fmt.Sprintf("%s_%s_%s", pod.Namespace, pod.Name, c.Name)

		filter := filters.NewArgs()
		filter.Add("name", containerName)
		containers, err := d.client.ContainerList(ctx, types.ContainerListOptions{
			All:     true,
			Filters: filter,
		})
		if err != nil || len(containers) == 0 {
			allRunning = false
			status.Containers = append(status.Containers, api.ContainerStatus{
				Name:  c.Name,
				State: "unknown",
			})
			continue
		}

		cont := containers[0]
		state := "unknown"
		if cont.State == "running" {
			state = "running"
		} else if cont.State == "exited" {
			state = "exited"
			allRunning = false
		} else {
			allRunning = false
		}

		status.Containers = append(status.Containers, api.ContainerStatus{
			Name:        c.Name,
			State:       state,
			ContainerID: cont.ID,
			Image:       cont.Image,
		})
	}

	if allRunning && len(pod.Spec.Containers) > 0 {
		status.Phase = api.PodRunning
	} else if len(status.Containers) > 0 {
		status.Phase = api.PodPending
	}

	return status, nil
}

// GetContainerLogs 获取容器日志
func (d *DockerRuntime) GetContainerLogs(pod *api.Pod, containerName string, tail int) (string, error) {
	ctx := context.Background()

	fullContainerName := fmt.Sprintf("%s_%s_%s", pod.Namespace, pod.Name, containerName)

	filter := filters.NewArgs()
	filter.Add("name", fullContainerName)
	containers, err := d.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filter,
	})
	if err != nil || len(containers) == 0 {
		return "", fmt.Errorf("container not found")
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	logs, err := d.client.ContainerLogs(ctx, containers[0].ID, options)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	output, err := io.ReadAll(logs)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// pullImage 拉取镜像
func (d *DockerRuntime) pullImage(ctx context.Context, image string) error {
	// 检查镜像是否存在
	_, _, err := d.client.ImageInspectWithRaw(ctx, image)
	if err == nil {
		return nil // 镜像已存在
	}

	// 拉取镜像
	reader, err := d.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// 等待拉取完成
	io.Copy(io.Discard, reader)
	return nil
}

// ListContainers 列出所有 minik8s 容器
func (d *DockerRuntime) ListContainers() ([]types.Container, error) {
	ctx := context.Background()
	filter := filters.NewArgs()
	filter.Add("label", "minik8s.io/pod")
	return d.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filter,
	})
}

// Close 关闭连接
func (d *DockerRuntime) Close() error {
	return d.client.Close()
}

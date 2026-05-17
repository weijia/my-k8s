package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/weijia/my-k8s/pkg/api"
	"github.com/weijia/my-k8s/pkg/runtime"
	"github.com/weijia/my-k8s/pkg/scheduler"
	"github.com/weijia/my-k8s/pkg/storage"
)

var (
	bindAddr    string
	dbPath      string
	enableCORS  bool
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the MiniK8s control plane server",
	Run:   runServer,
}

func init() {
	serverCmd.Flags().StringVar(&bindAddr, "bind", ":8080", "Address to bind to")
	serverCmd.Flags().StringVar(&dbPath, "db", "minik8s.db", "Path to SQLite database")
	serverCmd.Flags().BoolVar(&enableCORS, "cors", true, "Enable CORS")
}

func runServer(cmd *cobra.Command, args []string) {
	log.Printf("Starting MiniK8s Server on %s", bindAddr)

	// 初始化存储
	store, err := storage.NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// 初始化 Docker 运行时
	docker, err := runtime.NewDockerRuntime()
	if err != nil {
		log.Fatalf("Failed to initialize Docker runtime: %v", err)
	}
	defer docker.Close()

	// 初始化调度器
	sched := scheduler.NewScheduler(store)

	// 创建 Gin 路由器
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// 添加中间件
	if enableCORS {
		router.Use(corsMiddleware())
	}

	// 健康检查
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API 路由
	apiGroup := router.Group("/api/v1")
	{
		// Pods
		apiGroup.POST("/namespaces/:namespace/pods", createPodHandler(store, docker, sched))
		apiGroup.GET("/namespaces/:namespace/pods", listPodsHandler(store))
		apiGroup.GET("/namespaces/:namespace/pods/:name", getPodHandler(store))
		apiGroup.DELETE("/namespaces/:namespace/pods/:name", deletePodHandler(store, docker))
		apiGroup.GET("/namespaces/:namespace/pods/:name/logs", getPodLogsHandler(store, docker))

		// Nodes
		apiGroup.GET("/nodes", listNodesHandler(store))
		apiGroup.GET("/nodes/:name", getNodeHandler(store))
	}

	// 节点注册
	router.POST("/api/v1/nodes/register", registerNodeHandler(store))
	router.POST("/api/v1/nodes/:name/heartbeat", heartbeatHandler(store))

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    bindAddr,
		Handler: router,
	}

	// 启动服务器
	go func() {
		log.Printf("MiniK8s Server listening on %s", bindAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// Handlers

func createPodHandler(store *storage.Store, docker *runtime.DockerRuntime, sched *scheduler.Scheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		if namespace == "" {
			namespace = "default"
		}

		var pod api.Pod
		if err := c.ShouldBindJSON(&pod); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// 设置默认值
		if pod.Name == "" {
			c.JSON(400, gin.H{"error": "pod name is required"})
			return
		}
		if len(pod.Spec.Containers) == 0 {
			c.JSON(400, gin.H{"error": "at least one container is required"})
			return
		}
		if pod.Namespace == "" {
			pod.Namespace = namespace
		}

		// 设置类型元数据
		pod.APIVersion = "v1"
		pod.Kind = "Pod"
		pod.Status.Phase = api.PodPending

		// 调度 Pod
		nodeName, err := sched.Schedule(&pod)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("scheduling failed: %v", err)})
			return
		}

		// 如果不是本机节点，需要通过 RPC 或 Agent 创建
		// MVP 简化：仅支持本机
		if nodeName != "localhost" && nodeName != getHostname() {
			c.JSON(400, gin.H{"error": fmt.Sprintf("MVP only supports local pods, scheduled to: %s", nodeName)})
			return
		}

		// 创建容器
		if err := docker.CreatePod(&pod); err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("failed to create pod: %v", err)})
			return
		}

		// 保存到存储
		if err := store.SavePod(&pod); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, pod)
	}
}

func listPodsHandler(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		pods, err := store.ListPods(namespace)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, api.PodList{
			TypeMeta: api.TypeMeta{APIVersion: "v1", Kind: "PodList"},
			Items:    pods,
		})
	}
}

func getPodHandler(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		name := c.Param("name")
		pod, err := store.GetPod(namespace, name)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, pod)
	}
}

func deletePodHandler(store *storage.Store, docker *runtime.DockerRuntime) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		name := c.Param("name")

		pod, err := store.GetPod(namespace, name)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		// 删除容器
		if err := docker.DeletePod(pod); err != nil {
			log.Printf("Warning: failed to delete containers: %v", err)
		}

		// 从存储删除
		if err := store.DeletePod(namespace, name); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "pod deleted"})
	}
}

func getPodLogsHandler(store *storage.Store, docker *runtime.DockerRuntime) gin.HandlerFunc {
	return func(c *gin.Context) {
		namespace := c.Param("namespace")
		name := c.Param("name")
		container := c.DefaultQuery("container", "")
		tail := c.DefaultQuery("tail", "100")
		tailLines, _ := strconv.Atoi(tail)

		pod, err := store.GetPod(namespace, name)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		if container == "" && len(pod.Spec.Containers) > 0 {
			container = pod.Spec.Containers[0].Name
		}

		logs, err := docker.GetContainerLogs(pod, container, tailLines)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.String(200, logs)
	}
}

func listNodesHandler(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodes, err := store.ListNodes()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, api.NodeList{
			TypeMeta: api.TypeMeta{APIVersion: "v1", Kind: "NodeList"},
			Items:    nodes,
		})
	}
}

func getNodeHandler(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		node, err := store.GetNode(name)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, node)
	}
}

func registerNodeHandler(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name    string            `json:"name"`
			Address string            `json:"address"`
			CPU     string            `json:"cpu"`
			Memory  string            `json:"memory"`
			Labels  map[string]string `json:"labels"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		node := &api.Node{
			TypeMeta: api.TypeMeta{APIVersion: "v1", Kind: "Node"},
			ObjectMeta: api.ObjectMeta{
				Name:   req.Name,
				Labels: req.Labels,
			},
			Status: api.NodeStatus{
				Capacity: map[string]string{
					"cpu":    req.CPU,
					"memory": req.Memory,
				},
				Allocatable: map[string]string{
					"cpu":    req.CPU,
					"memory": req.Memory,
				},
				Conditions: []api.NodeCondition{
					{Type: api.NodeReady, Status: "True"},
				},
				Addresses: []api.NodeAddress{
					{Type: "InternalIP", Address: req.Address},
				},
			},
		}

		if err := store.SaveNode(node); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, node)
	}
}

func heartbeatHandler(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		node, err := store.GetNode(name)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		node.Status.Conditions = []api.NodeCondition{
			{Type: api.NodeReady, Status: "True"},
		}

		if err := store.SaveNode(node); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	}
}

// Middleware

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// Utils

func getHostname() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	return hostname
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

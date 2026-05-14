package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/weijia/my-k8s/pkg/api"
)

var (
	serverURL string
	nodeName  string
	nodeIP    string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Start the MiniK8s node agent",
	Run:   runAgent,
}

func init() {
	agentCmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Server URL")
	agentCmd.Flags().StringVar(&nodeName, "name", "", "Node name (default: hostname)")
	agentCmd.Flags().StringVar(&nodeIP, "ip", "", "Node IP (auto-detected if not specified)")
}

func runAgent(cmd *cobra.Command, args []string) {
	log.Printf("Starting MiniK8s Agent")
	log.Printf("Server: %s", serverURL)

	if nodeName == "" {
		nodeName, _ = os.Hostname()
	}
	if nodeName == "" {
		nodeName = "node-" + fmt.Sprintf("%d", time.Now().Unix())
	}

	if nodeIP == "" {
		nodeIP = getOutboundIP()
	}

	log.Printf("Node Name: %s", nodeName)
	log.Printf("Node IP: %s", nodeIP)

	// 注册节点
	if err := registerNode(); err != nil {
		log.Fatalf("Failed to register node: %v", err)
	}

	// 启动心跳
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Agent started successfully")

	for {
		select {
		case <-ticker.C:
			if err := sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}
		}
	}
}

func registerNode() error {
	req := map[string]interface{}{
		"name":    nodeName,
		"address": nodeIP,
		"cpu":     "4",
		"memory":  "8192Mi",
		"labels": map[string]string{
			"kubernetes.io/os":   runtime.GOOS,
			"kubernetes.io/arch": runtime.GOARCH,
		},
	}

	data, _ := json.Marshal(req)
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/nodes/register", serverURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	var node api.Node
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return err
	}

	log.Printf("Node registered: %s", node.Name)
	return nil
}

func sendHeartbeat() error {
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/v1/nodes/%s/heartbeat", serverURL, nodeName),
		nil,
	)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}

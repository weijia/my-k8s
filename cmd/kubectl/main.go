package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/weijia/my-k8s/pkg/api"
	"gopkg.in/yaml.v3"
)

var (
	serverURL string
	namespace string
	output    string
)

var rootCmd = &cobra.Command{
	Use:   "kubectl",
	Short: "MiniK8s CLI",
	Long:  `Command line interface for MiniK8s`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", getEnv("MINIK8S_SERVER", "http://localhost:8080"), "Server URL")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Namespace")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Output format (json|yaml|wide)")

	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(logsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// create 命令
var createCmd = &cobra.Command{
	Use:   "create -f <filename>",
	Short: "Create a resource from a file",
	Run:   runCreate,
}

var createFile string

func init() {
	createCmd.Flags().StringVarP(&createFile, "filename", "f", "", "Filename to use to create the resource")
	createCmd.MarkFlagRequired("filename")
}

func runCreate(cmd *cobra.Command, args []string) {
	if createFile == "" {
		fmt.Println("Error: --filename is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(createFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// 解析 YAML
	var pod api.Pod
	if err := yaml.Unmarshal(data, &pod); err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// 发送请求
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods", serverURL, namespace)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Printf("Error creating pod: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Printf("pod/%s created\n", pod.Name)
}

// get 命令
var getCmd = &cobra.Command{
	Use:   "get <resource>",
	Short: "Display one or many resources",
	Run:   runGet,
}

func runGet(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: resource type is required")
		os.Exit(1)
	}

	resource := args[0]

	switch resource {
	case "pods", "pod", "po":
		getPods()
	case "nodes", "node", "no":
		getNodes()
	default:
		fmt.Printf("Unknown resource type: %s\n", resource)
		os.Exit(1)
	}
}

func getPods() {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods", serverURL, namespace)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var podList api.PodList
	if err := json.NewDecoder(resp.Body).Decode(&podList); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	if output == "json" {
		data, _ := json.MarshalIndent(podList, "", "  ")
		fmt.Println(string(data))
		return
	}

	if output == "yaml" {
		data, _ := yaml.Marshal(podList)
		fmt.Println(string(data))
		return
	}

	// 表格输出
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tNODE\tAGE")
	for _, pod := range podList.Items {
		age := "<unknown>"
		if !pod.CreationTimestamp.IsZero() {
			age = formatDuration(pod.CreationTimestamp)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", pod.Name, pod.Status.Phase, pod.Spec.NodeName, age)
	}
	w.Flush()
}

func getNodes() {
	url := fmt.Sprintf("%s/api/v1/nodes", serverURL)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var nodeList api.NodeList
	if err := json.NewDecoder(resp.Body).Decode(&nodeList); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		os.Exit(1)
	}

	if output == "json" {
		data, _ := json.MarshalIndent(nodeList, "", "  ")
		fmt.Println(string(data))
		return
	}

	if output == "yaml" {
		data, _ := yaml.Marshal(nodeList)
		fmt.Println(string(data))
		return
	}

	// 表格输出
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tROLES\tAGE\tVERSION")
	for _, node := range nodeList.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == api.NodeReady && cond.Status == "True" {
				status = "Ready"
				break
			}
		}
		age := "<unknown>"
		if !node.CreationTimestamp.IsZero() {
			age = formatDuration(node.CreationTimestamp)
		}
		fmt.Fprintf(w, "%s\t%s\t<none>\t%s\tv0.1.0\n", node.Name, status, age)
	}
	w.Flush()
}

// delete 命令
var deleteCmd = &cobra.Command{
	Use:   "delete <resource> <name>",
	Short: "Delete resources",
	Run:   runDelete,
}

func runDelete(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Error: resource type and name are required")
		os.Exit(1)
	}

	resource := args[0]
	name := args[1]

	switch resource {
	case "pods", "pod", "po":
		deletePod(name)
	default:
		fmt.Printf("Unknown resource type: %s\n", resource)
		os.Exit(1)
	}
}

func deletePod(name string) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", serverURL, namespace, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Printf("pod/%s deleted\n", name)
}

// logs 命令
var logsCmd = &cobra.Command{
	Use:   "logs <pod-name>",
	Short: "Print the logs for a container in a pod",
	Run:   runLogs,
}

var logsContainer string
var logsTail int

func init() {
	logsCmd.Flags().StringVar(&logsContainer, "container", "", "Container name")
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show")
}

func runLogs(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: pod name is required")
		os.Exit(1)
	}

	name := args[0]
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/logs?tail=%d", serverURL, namespace, name, logsTail)
	if logsContainer != "" {
		url += "&container=" + logsContainer
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: %s\n", string(body))
		os.Exit(1)
	}

	io.Copy(os.Stdout, resp.Body)
}

// 工具函数

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func formatDuration(t interface{}) string {
	// 简化实现
	return "<unknown>"
}

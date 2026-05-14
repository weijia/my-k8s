.PHONY: all build build-server build-kubectl clean test run

# 默认目标
all: build

# 构建所有二进制文件
build: build-server build-kubectl

# 构建 minik8s server
build-server:
	go build -o bin/minik8s ./cmd/minik8s

# 构建 kubectl
build-kubectl:
	go build -o bin/kubectl ./cmd/kubectl

# 清理构建产物
clean:
	rm -rf bin/
	rm -f minik8s.db

# 运行测试
test:
	go test -v ./...

# 下载依赖
deps:
	go mod download
	go mod tidy

# 运行 server（开发模式）
run-server: build-server
	./bin/minik8s server --bind :8080

# 运行 agent（开发模式）
run-agent: build-server
	./bin/minik8s agent --server http://localhost:8080

# 跨平台构建
build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/minik8s-linux ./cmd/minik8s
	GOOS=linux GOARCH=amd64 go build -o bin/kubectl-linux ./cmd/kubectl

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o bin/minik8s-darwin ./cmd/minik8s
	GOOS=darwin GOARCH=amd64 go build -o bin/kubectl-darwin ./cmd/kubectl

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/minik8s.exe ./cmd/minik8s
	GOOS=windows GOARCH=amd64 go build -o bin/kubectl.exe ./cmd/kubectl

# 构建所有平台
build-all: build-linux build-darwin build-windows

# 安装到系统
install: build
	cp bin/minik8s /usr/local/bin/
	cp bin/kubectl /usr/local/bin/

# 卸载
uninstall:
	rm -f /usr/local/bin/minik8s
	rm -f /usr/local/bin/kubectl

BINARY  := vmctl
BIN_DIR := bin
PKG     := ./cmd/vmctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X gitee.com/normalcoder/vmctl/internal/cli.Version=$(VERSION)

.PHONY: build install run test vet fmt check clean

build: ## 构建到 bin/vmctl
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

install: ## 安装到 GOPATH/bin
	go install -ldflags "$(LDFLAGS)" $(PKG)

run: build ## 构建并运行
	./$(BIN_DIR)/$(BINARY)

test: ## 运行测试
	go test ./...

vet: ## 静态检查
	go vet ./...

fmt: ## 格式化代码
	gofmt -w .

check: fmt vet test ## 格式化 + 检查 + 测试

clean: ## 清理构建产物
	rm -rf $(BIN_DIR)

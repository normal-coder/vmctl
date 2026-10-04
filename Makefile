BINARY  := vmctl
BIN_DIR := bin
PKG     := ./cmd/vmctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X gitee.com/normalcoder/vmctl/internal/cli.Version=$(VERSION)

# 发布相关（延迟求值：只有 next-version/release 才触发 git-cliff）
# 首发无 tag 时 git-cliff 返回 0.1.0（无 v 前缀），统一补 v 以匹配 CI 的 v* 触发
RAW_NEXT_VERSION = $(shell git-cliff --bumped-version 2>/dev/null)
NEXT_VERSION     = $(if $(filter v%,$(RAW_NEXT_VERSION)),$(RAW_NEXT_VERSION),$(if $(RAW_NEXT_VERSION),v$(RAW_NEXT_VERSION)))

.PHONY: build install run test vet fmt check clean next-version release snapshot

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
	rm -rf $(BIN_DIR) dist

next-version: ## 预览下一个版本号与待发布提交（需 git-cliff）
	@command -v git-cliff >/dev/null 2>&1 || { echo "error: git-cliff not found (brew install git-cliff)" >&2; exit 1; }
	@echo "current: $(VERSION)"
	@echo "next:    $(NEXT_VERSION)"
	@echo "---"
	@git-cliff --unreleased

release: ## 发布：预览 → y 确认 → 更新 CHANGELOG → GPG 签名 tag → push 触发 CI
	@command -v git-cliff >/dev/null 2>&1 || { echo "error: git-cliff not found (brew install git-cliff)" >&2; exit 1; }
	$(eval TAG_IN := $(if $(VERSION_OVERRIDE),$(VERSION_OVERRIDE),$(NEXT_VERSION)))
	@if [ -z "$(TAG_IN)" ]; then echo "error: could not determine next version" >&2; exit 1; fi
	$(eval TAG := v$(patsubst v%,%,$(TAG_IN)))
	@echo "==> releasing $(TAG)"
	@echo ""
	@git-cliff --unreleased --tag $(TAG)
	@echo ""
	@printf "confirm release $(TAG)? [y/N] "; read ans; [ "$$ans" = "y" ] || { echo "aborted"; exit 1; }
	@git-cliff --unreleased --tag $(TAG) -o /tmp/vmctl-tag-msg.txt
	@if [ -f CHANGELOG.md ]; then \
		git-cliff --unreleased --tag $(TAG) --prepend CHANGELOG.md; \
	else \
		git-cliff --tag $(TAG) -o CHANGELOG.md; \
	fi
	@git add CHANGELOG.md
	@git commit -m "chore(release): $(TAG)"
	@git tag -s $(TAG) -F /tmp/vmctl-tag-msg.txt
	@git push origin main --follow-tags
	@echo "==> pushed $(TAG), CI will build and publish the release"

snapshot: ## 本地多平台构建快照（goreleaser，不发布、不需要 token）
	goreleaser build --snapshot --clean

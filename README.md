# vmctl

vmctl 是一个参考 `prlctl` 命令风格设计的 VMware 虚拟机控制 CLI，使用 Go 编写。

首版基于本地 `vmrun`（VMware Fusion / Workstation），架构上通过统一的 Driver 接口预留了
远程 vSphere 后端（govomi）的扩展位置。

## 构建

```bash
make build          # 构建到 bin/vmctl（版本号自动注入 git 短哈希）
make check          # 格式化 + go vet + 测试
make install        # 安装到 GOPATH/bin
```

或直接使用 go 命令：

```bash
go build -o bin/vmctl ./cmd/vmctl
```

## 快速上手

```bash
# 列出所有虚拟机
vmctl list
vmctl list --json

# 查看虚拟机详情（名称、UUID、Tools 状态、Guest IP 等）
vmctl info <vm>

# 电源管理
vmctl start <vm> [--nogui]
vmctl stop <vm> [--hard]
vmctl suspend <vm>
vmctl resume <vm>
vmctl pause <vm>
vmctl reset <vm> [--hard]

# 克隆（默认 linked clone，--full 为完整拷贝；源须已关机）
vmctl clone <vm> --name <new-name> [--full] [--snapshot <name>] [--path <dest>]

# 从现有 VM 派生新 VM（vmrun 后端不支持从零创建裸 VM）
vmctl create <name> --from <vm> [--memory 4096] [--cpus 4] [--full]

# 修改配置（须已关机）
vmctl set <vm> --memory 4096 --cpus 4 [--name <new-name>]

# 删除虚拟机及其全部文件（不可恢复）
vmctl delete <vm>
```

`<vm>` 支持多种引用方式：

- 显示名（不区分大小写）：`vmctl stop my-vm`
- 名称的唯一子串：`vmctl stop ubuntu`
- UUID 或 UUID 前缀：`vmctl info 564d1156`
- vmx 文件路径 / .vmwarevm 目录路径

## 全局选项

| 选项 | 说明 |
|---|---|
| `--json` | 以 JSON 输出（便于脚本处理） |
| `--backend` | 后端驱动，默认 `vmrun`（预留 vsphere） |
| `--vmrun` | 指定 vmrun 路径（默认自动探测） |

环境变量 `VMCTL_VMRUN` 同样可以指定 vmrun 路径。

## 架构

```
cmd/vmctl          入口
internal/cli       cobra 命令树
internal/output    表格 / JSON 渲染
internal/driver    统一 Driver 接口 + 注册表
  └── vmrun/       本地 vmrun 驱动（解析 Fusion vmInventory + 调用 vmrun）
internal/model     与后端无关的 VM 模型
```

## 路线图

- [x] M1：骨架 + Driver 接口 + list / info / 电源操作 + JSON 输出
- [x] M2：create / clone / set / delete
- [ ] M3：snapshot 全套 + guest 内 exec
- [ ] M4：vsphere 后端（govomi）+ profile 配置
- [ ] M5：shell 补全、体验打磨

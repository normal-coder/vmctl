# vmctl

vmctl 是一个参考 `prlctl` 命令风格设计的 VMware 虚拟机控制 CLI，使用 Go 编写。

通过统一的 Driver 接口支持两种后端：

- **vmrun**（默认）：本地 VMware Fusion / Workstation；
- **vsphere**：远程 vCenter / ESXi（govmomi）。

配置文件支持多 profile（一条命令切换本地/远程环境），界面文案支持中英双语。

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

# 电源管理（start 返回即代表 VM 已加电，不等待 guest 就绪）
vmctl start <vm> [--nogui]
vmctl stop <vm> [--hard]
vmctl suspend <vm>
vmctl resume <vm>
vmctl pause <vm>
vmctl reset <vm> [--hard]

# 克隆（默认 linked clone，--full 为完整拷贝；vmrun 要求源已关机，vsphere 开机也可）
vmctl clone <vm> --name <new-name> [--full] [--snapshot <name>] [--path <dest>]

# 从现有 VM 派生新 VM（vmrun 后端不支持从零创建裸 VM）
vmctl create <name> --from <vm> [--memory 4096] [--cpus 4] [--full]

# 修改配置（须已关机）
vmctl set <vm> --memory 4096 --cpus 4 [--name <new-name>]

# 删除虚拟机及其全部文件（不可恢复）
vmctl delete <vm>

# 快照管理（revert 会一并恢复快照时的电源状态）
vmctl snapshot list <vm> [--tree]
vmctl snapshot create <vm> <name>
vmctl snapshot delete <vm> <name> [--children]
vmctl snapshot revert <vm> <name>       # 须已关机

# 在 guest 内执行命令（经 /bin/sh，捕获输出，透传退出码）
# 需要 VM 已开机且 guest 内装有 VMware Tools
vmctl exec <vm> -- 'df -h | grep /'
vmctl exec <vm> -u root -- systemctl restart nginx

# 查看 guest IP（脚本友好，输出裸 IP；需要 Tools）
vmctl ip <vm> [--wait]

# 通过 SSH 进入 guest（需 guest 开启 sshd）
vmctl shell <vm> [--user u] [--port 22] [-i key] [--wait] [--dry-run]
vmctl shell <vm> -- -v                    # -- 之后透传给 ssh
```

`<vm>` 支持多种引用方式：

- 显示名（不区分大小写）：`vmctl stop my-vm`
- 名称的唯一子串：`vmctl stop ubuntu`
- UUID 或 UUID 前缀：`vmctl info 564d1156`
- vmx 文件路径 / .vmwarevm 目录路径（vmrun）
- MoRef（`vm-42` / `VirtualMachine:vm-42`）与 inventory path（vsphere）

## 配置文件

默认读取 `~/.config/vmctl/config.yaml`（`macOS` 为 `~/Library/Application Support/vmctl/`），
可用环境变量 `VMCTL_CONFIG` 指向自定义路径（指向不存在的文件会直接报错）：

```yaml
default: lab
profiles:
  lab:
    backend: vsphere
    endpoint: https://vcenter.example.com
    user: administrator@vsphere.local
    password_env: VMCTL_VCENTER_PASSWORD   # 或直接写 password: 明文
    insecure: true                         # 自签证书时跳过校验
  local:
    backend: vmrun
    # vmrun: /Applications/VMware Fusion.app/.../vmrun   # 可选
```

- `default` 指定无 `--profile` 时使用的 profile；文件缺省时不报错（行为与无配置一致）。
- 字段融合优先级：**显式 CLI flag > profile 字段 > 内置默认**（backend 默认 `vmrun`）。
- 密码取值：`password` 优先，为空则用 `password_env` 查环境变量（未设置会报错）。
- 未知字段会被严格模式拒绝，防止拼写错误静默生效。

```bash
vmctl --profile lab list        # 用指定 profile
vmctl --profile local list      # 一条命令切回本地
```

## 界面语言

默认中文，`--lang en` 切换英文（或设置环境变量 `VMCTL_LANG`）：

```bash
vmctl list --help               # 中文帮助
vmctl --lang en list --help     # 英文帮助
```

语言在启动时预扫描 `--lang` 参数确定，早于帮助文案构建，因此 help 与错误信息都会切换。

## 全局选项

| 选项 | 说明 |
|---|---|
| `--json` | 以 JSON 输出（便于脚本处理） |
| `--backend` | 后端驱动，默认 `vmrun` |
| `--profile` | 使用的配置 profile（缺省取配置文件的 `default`） |
| `--lang` | 界面语言：`zh` / `en`，默认 `zh` |
| `--vmrun` | 指定 vmrun 路径（默认自动探测） |

环境变量：`VMCTL_VMRUN`（vmrun 路径）、`VMCTL_CONFIG`（配置文件路径）、
`VMCTL_LANG`（界面语言）。

## vSphere 后端

在 profile 中配置 `backend: vsphere` 后即可直接使用全部命令：

```bash
vmctl --profile lab list
vmctl --profile lab info my-vm
vmctl --profile lab start my-vm
vmctl --profile lab snapshot create my-vm before-upgrade
vmctl --profile lab exec my-vm -- 'df -h | grep /'
```

虚拟机引用（resolve）按以下顺序匹配：MoRef（`vm-42` / `VirtualMachine:vm-42`）
→ 精确名（大小写不敏感）→ UUID 前缀（≥8 位十六进制）→ 名称唯一子串 →
inventory path（如 `DC0/vm/my-vm`）。0 命中报错并列出可用名称，多命中报歧义。

与 vmrun 后端的行为差异：

| 场景 | vmrun | vsphere |
|---|---|---|
| 克隆时源 VM 状态 | 必须关机 | 开机也可（linked clone 基于快照） |
| `clone`/`create` 返回值 | vmx 文件路径 | inventory path（如 `DC0/vm/my-vm`） |
| `pause` | 支持 | 不支持（用 `suspend` 挂起到磁盘） |
| `snapshot revert` | 恢复快照时电源状态 | 回退后不自动开机（`suppressPowerOn`） |
| `create` 无 `--from` | 报错 | `ErrNotSupported`（无法创建裸 VM，请用 `--from`） |

> **验证程度**：vsphere 后端已通过 vcsim（govmomi 模拟器）集成测试覆盖库存、电源、
> 克隆、快照与文案切换；**真实 vCenter / ESXi 尚未实测**，首次接入时建议先在测试
> 环境冒烟（软关机、linked clone、exec 输出与退出码、模板克隆）。

## 架构

```
cmd/vmctl          入口
internal/cli       cobra 命令树
internal/output    表格 / JSON 渲染
internal/config    配置文件加载与 profile 解析（YAML，严格模式）
internal/i18n      轻量消息目录（中英切换，key 对齐测试）
internal/driver    统一 Driver 接口 + 注册表
  ├── vmrun/       本地 vmrun 驱动（解析 Fusion vmInventory + 调用 vmrun）
  └── vsphere/     远程 vSphere 驱动（govmomi：库存/电源/克隆/快照/guest ops）
internal/model     与后端无关的 VM 模型
```

## 路线图

- [x] M1：骨架 + Driver 接口 + list / info / 电源操作 + JSON 输出
- [x] M2：create / clone / set / delete
- [x] M3：snapshot 全套 + guest exec + ip / shell
- [x] M4：vsphere 后端（govmomi）+ profile 配置 + 多语言文案
- [ ] M5：vmcli 同名快照 fallback、vmrun 存量英文错误汉化回填、shell 补全、体验打磨

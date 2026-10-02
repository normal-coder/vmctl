package i18n

// zhCatalog is the Chinese message directory. Keys must stay in sync
// with enCatalog (enforced by TestCatalogParity).
var zhCatalog = map[string]string{
	// root
	"root.short": "控制 VMware 虚拟机",
	"root.long":  "vmctl 是参考 prlctl 命令风格的 VMware 虚拟机控制工具。",

	// global flags
	"flag.json":    "以 JSON 格式输出",
	"flag.vmrun":   "vmrun 可执行文件路径（默认自动探测）",
	"flag.backend": "使用的后端驱动",
	"flag.lang":    "界面语言：zh 或 en（默认 zh）",
	"flag.profile": "使用的配置 profile（缺省取配置文件的 default）",

	// config / profile
	"err.config.notExist":      "找不到配置文件：%s",
	"err.config.read":          "读取配置文件失败：%s",
	"err.config.parse":         "解析配置文件失败：%s",
	"err.config.profile":       "找不到 profile %q，可用的有：%s",
	"err.config.profileNoList": "找不到 profile %q，且配置中未定义任何 profile",
	"err.config.envEmpty":      "环境变量 %s 未设置或为空，无法读取密码",

	// errors
	"err.prefix":       "错误",
	"err.notSupported": "此操作该后端不支持",

	// list / info
	"cmd.list.short": "列出虚拟机",
	"cmd.info.short": "显示虚拟机详细信息",

	// power
	"cmd.start.short":   "启动或恢复虚拟机",
	"cmd.stop.short":    "关闭虚拟机电源",
	"cmd.suspend.short": "将虚拟机挂起到磁盘",
	"cmd.resume.short":  "恢复挂起或暂停的虚拟机",
	"cmd.pause.short":   "暂停正在运行的虚拟机",
	"cmd.reset.short":   "重置虚拟机",
	"flag.hard":         "强制执行（跳过客户机正常关机）",
	"flag.nogui":        "启动时不打开窗口",

	// clone
	"cmd.clone.short": "克隆虚拟机",
	"cmd.clone.long": `克隆虚拟机。默认创建链接克隆（速度快，会在源虚拟机上
保留一个基线快照）。使用 --full 创建独立完整拷贝。
源虚拟机必须已关机。`,
	"flag.clone.name": "新虚拟机名称（必填）",
	"flag.full":       "创建独立完整拷贝，而非链接克隆",
	"flag.snapshot":   "克隆基线快照名称",
	"flag.path":       "目标目录或 .vmx 路径",

	// create
	"cmd.create.short": "从现有虚拟机派生新虚拟机",
	"cmd.create.long": `通过克隆源虚拟机来交付一台新虚拟机。
vmrun 后端不支持从零创建裸虚拟机（Fusion 未提供
createVM 命令）。`,
	"flag.from":   "克隆源虚拟机（必填）",
	"flag.memory": "新虚拟机内存，单位 MB（0 = 沿用源）",
	"flag.cpus":   "新虚拟机 CPU 数（0 = 沿用源）",

	// set
	"cmd.set.short":   "修改虚拟机配置",
	"cmd.set.long":    "修改已关机虚拟机的内存、CPU 数或显示名称。",
	"err.set.flags":   "至少指定 --name、--memory、--cpus 之一",
	"flag.set.name":   "新的显示名称",
	"flag.set.memory": "内存，单位 MB",
	"flag.set.cpus":   "CPU 数",

	// delete
	"cmd.delete.short": "永久删除虚拟机及其全部文件",
	"cmd.delete.long":  "永久删除虚拟机，包括磁盘上的全部文件。此操作不可恢复。",

	// snapshot
	"cmd.snapshot.short":        "管理虚拟机快照",
	"cmd.snapshot.long":         "快照操作：列出、创建、删除与回退。\n回退要求虚拟机已关机。",
	"cmd.snapshot.list.short":   "列出虚拟机的快照",
	"cmd.snapshot.create.short": "创建快照",
	"cmd.snapshot.delete.short": "删除快照",
	"cmd.snapshot.delete.long":  "删除快照。默认将子快照重新挂到其父级；\n使用 --children 可一并删除子快照。",
	"cmd.snapshot.revert.short": "将虚拟机回退到指定快照",
	"flag.tree":                 "显示快照层级",
	"flag.children":             "同时删除子快照",

	// exec
	"cmd.exec.short": "在客户机内执行命令",
	"cmd.exec.long": `通过 VMware Tools 在客户机内执行命令。命令经 /bin/sh
运行，输出会被打印，客户机退出码即为 vmctl 退出码。
要使用 shell 语法请整体加引号，例如：
  vmctl exec my-vm -- 'df -h | grep /'
客户机凭据会出现在 vmrun 的命令行上；建议用
VMCTL_GUEST_PASSWORD 环境变量代替 --password，
避免密码进入 shell 历史。`,
	"flag.exec.user":     "客户机用户名（默认：当前主机用户）",
	"flag.exec.password": "客户机密码（或设置 VMCTL_GUEST_PASSWORD）",

	// shell
	"cmd.shell.short": "通过 SSH 进入客户机",
	"cmd.shell.long": `经 VMware Tools 解析客户机 IP 并启动 ssh。
客户机须运行可访问的 sshd，支持密钥或密码认证。
-- 之后的选项会透传给 ssh，例如：
  vmctl shell my-vm -- -v
使用 --dry-run 可只打印 ssh 命令而不实际连接。`,
	"err.shell.args":      "只接受 1 个参数，收到 %d 个",
	"flag.shell.user":     "登录用户（默认：当前主机用户）",
	"flag.shell.port":     "ssh 端口",
	"flag.shell.identity": "ssh 私钥文件",
	"flag.wait":           "等待客户机上报 IP",
	"flag.dry-run":        "只打印 ssh 命令而不实际连接",

	// ip
	"cmd.ip.short": "打印客户机 IP 地址",
	"cmd.ip.long":  "打印 VMware Tools 上报的客户机 IP —— 便于脚本使用：\nvmctl ip my-vm",

	// version
	"cmd.version.short": "显示 vmctl 版本",
}

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
	"flag.vmcli":   "vmcli 可执行文件路径（默认自动探测，需要 Fusion 13.5+）",
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

	// resolve (shared reference resolution wording)
	"err.resolve.empty":          "引用不能为空",
	"err.resolve.notFound":       "找不到虚拟机 %q（可用：%s）",
	"err.resolve.ambiguous":      "引用 %q 有歧义，匹配到 %d 台虚拟机",
	"err.resolve.ambiguousNames": "引用 %q 有歧义，匹配到：%s",

	// vmcli (snapshot uid fallback)
	"err.vmcli.notFound": "找不到 vmcli（需要 VMware Fusion 13.5+，可用 --vmcli 或 VMCTL_VMCLI 指定路径）",

	// vmrun backend
	"err.name.empty":           "名称不能为空",
	"err.name.invalid":         "无效的名称 %q：不能包含 /、\\、引号或控制字符",
	"err.name.exists":          "已存在名为 %q 的虚拟机",
	"err.vmrun.notFound":       "找不到 vmrun（请安装 VMware Fusion/Workstation 或设置 VMCTL_VMRUN）",
	"err.vmrun.runningFor":     "无法执行 %s：虚拟机正在运行，请先关机",
	"err.vmrun.notRunningFor":  "无法执行 %s：虚拟机未在运行，请先开机",
	"err.vmrun.powerOnTimeout": "等待 %q 开机超时",
	"err.vmrun.offUseStart":    "%q 处于关机状态，请改用 start",
	"err.vmrun.exitZero":       "vmrun 以退出码 0 结束但报告了错误",
	"err.dest.exists":          "目标路径已存在：%s",
	"err.dest.notDir":          "目标不是目录：%s",
	"err.dest.notEmpty":        "目标目录非空：%s",
	"err.clone.regFail":        "已克隆到 %s，但库存注册失败：%w",
	"err.clone.configFail":     "已克隆到 %s，但克隆后配置失败：%w",
	"err.clone.noFrom":         "此后端无法创建裸虚拟机，请用 --from 指定源虚拟机",
	"err.set.memoryPositive":   "内存必须大于 0 MB",
	"err.set.cpusPositive":     "CPU 数必须大于 0",
	"err.set.renameFail":       "已更新 %s，但库存重命名失败：%w",
	"err.delete.cleanupFail":   "已删除，但库存清理失败：%w",
	"err.vmx.utf16":            "%s：不支持编辑 UTF-16 编码的 vmx 文件",
	"err.snapshot.invalidName": "无效的快照名：%w",

	// output
	"output.noVMs":       "未找到虚拟机。",
	"output.noSnapshots": "%s 没有快照。",
	"output.ok":          "成功",

	// vsphere backend
	"err.vsphere.endpoint":         "vsphere 后端缺少 endpoint，请在 profile 中配置",
	"err.vsphere.user":             "vsphere 后端缺少登录用户名",
	"err.vsphere.connect":          "连接 %s 失败：%s",
	"err.vsphere.notRunning":       "虚拟机未处于运行状态",
	"err.vsphere.stopTimeout":      "等待虚拟机关机超时（60 秒），客户机可能未安装或未运行 VMware Tools",
	"err.vsphere.offForResume":     "虚拟机已关机，无法恢复，请改用 start",
	"err.vsphere.noPause":          "vSphere 没有 pause 操作，可用 suspend 将虚拟机挂起到磁盘",
	"err.vsphere.mustBeOff":        "虚拟机必须处于关机状态",
	"err.vsphere.createNoFrom":     "vsphere 后端无法创建裸虚拟机，请用 --from 指定源虚拟机",
	"err.vsphere.linkedNoSnapshot": "链接克隆需要基线快照，但源虚拟机 %q 没有任何快照",
	"err.vsphere.cloneFolder":      "找不到克隆目标文件夹：%s",
	"err.vsphere.cloneNotFolder":   "克隆目标不是文件夹：%s",
	"err.vsphere.cloneResult":      "克隆任务未返回新虚拟机",
	"err.clone.name":               "必须指定新虚拟机名称",
	"err.vsphere.noIP":             "客户机尚未上报 IP 地址（VMware Tools 是否在运行？）",
	"err.exec.empty":               "命令不能为空",

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
	"cmd.snapshot.short":            "管理虚拟机快照",
	"cmd.snapshot.long":             "快照操作：列出、创建、删除与回退。\n回退要求虚拟机已关机。",
	"cmd.snapshot.list.short":       "列出虚拟机的快照",
	"cmd.snapshot.create.short":     "创建快照",
	"cmd.snapshot.delete.short":     "删除快照",
	"cmd.snapshot.delete.long":      "删除快照。默认将子快照重新挂到其父级；\n使用 --children 可一并删除子快照。",
	"cmd.snapshot.revert.short":     "将虚拟机回退到指定快照",
	"flag.tree":                     "显示快照层级",
	"flag.children":                 "同时删除子快照",
	"err.snapshot.needName":         "必须指定快照名",
	"err.snapshot.ambiguous":        "快照名 %q 有歧义，可用的引用：%s（用 <名字#uid> 指定其中一个）",
	"err.snapshot.ambiguousNoVmcli": "快照名 %q 有歧义，且无法通过 vmcli 列出候选；\n可用 --vmcli 指定 vmcli 路径以启用 <名字#uid> 引用",

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

	// completion
	"cmd.completion.short": "生成 shell 自动补全脚本",
	"cmd.completion.long": `为指定的 shell 生成 vmctl 自动补全脚本，
覆盖命令、全局 flag（--backend/--profile/--lang）
与虚拟机名。各 shell 的加载方式见子命令帮助。`,
	"cmd.completion.bash":       "生成 bash 补全脚本",
	"cmd.completion.zsh":        "生成 zsh 补全脚本",
	"cmd.completion.fish":       "生成 fish 补全脚本",
	"cmd.completion.powershell": "生成 powershell 补全脚本",
}

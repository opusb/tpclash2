package main

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/capability.h
const (
	CAP_NET_BIND_SERVICE = 10
	CAP_NET_ADMIN        = 12
	CAP_NET_RAW          = 13
)

const (
	ChainDockerUser = "DOCKER-USER" // https://docs.docker.com/network/packet-filtering-firewalls/#docker-on-a-router
)

const (
	InternalClashBinName = "xclash"
	InternalConfigName   = "xclash.yaml"
)

const logo = `
████████╗██████╗  ██████╗██╗      █████╗ ███████╗██╗  ██╗
╚══██╔══╝██╔══██╗██╔════╝██║     ██╔══██╗██╔════╝██║  ██║
   ██║   ██████╔╝██║     ██║     ███████║███████╗███████║
   ██║   ██╔═══╝ ██║     ██║     ██╔══██║╚════██║██╔══██║
   ██║   ██║     ╚██████╗███████╗██║  ██║███████║██║  ██║
   ╚═╝   ╚═╝      ╚═════╝╚══════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
`

const systmedTpl = `[Unit]
Description=Transparent proxy tool for Clash
After=network.target

[Service]
Type=simple
User=root
Restart=on-failure
ExecStart=/usr/local/bin/tpclash%s

RestartSec=10s
TimeoutStopSec=30s

[Install]
WantedBy=multi-user.target
`

const (
	installDir = "/usr/local/bin"
	systemdDir = "/etc/systemd/system"
)

const installedMessage = logo + `  👌 TPClash 安装完成, 您可以使用以下命令启动:
    - 启动服务: systemctl start tpclash
    - 停止服务: systemctl stop tpclash
    - 开启自启动: systemctl enable tpclash
    - 关闭自启动: systemctl disable tpclash
	- 查看日志: journalctl -fu tpclash
`

const uninstallMessage = `  
  ⚠️ 在卸载前请务必先停止 TPClash
  ⚠️ 如果尚未停止请按 Ctrl+c 终止卸载
  ⚠️ 本卸载程序将会在 30s 后继续执行卸载
`

const uninstalledMessage = logo + `  👌 TPClash 已卸载`

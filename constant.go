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

const (
	tracingNetworkName = "tpclash-tracing"
	lokiHostname       = "tpclash-loki"
	vectorHostname     = "tpclash-vector"

	lokiImage = "grafana/loki:2.8.0"
)

const installedMessage = logo + `  👌 TPClash 安装完成, 您可以使用以下命令启动:
     - 启动服务: systemctl start tpclash
     - 停止服务: systemctl stop tpclash
     - 重启服务: systemctl restart tpclash
     - 开启自启动: systemctl enable tpclash
     - 关闭自启动: systemctl disable tpclash
     - 查看日志: journalctl -fu tpclash
     - 重载服务配置: systemctl daemon-reload
`

const reinstallMessage = `
  ❗监测到您可能执行了重新安装, 重新启动前请执行重载服务配置.
`

const uninstallMessage = `  
  ❗️在卸载前请务必先停止 TPClash
  ❗️如果尚未停止请按 Ctrl+c 终止卸载
  ❗️本卸序将会在 30s 后继续执行卸载命令

`

const uninstalledMessage = logo + `  👌 TPClash 已卸载, 如有任何问题请开启 issue 或从 Telegram 讨论组反馈
     - 官方仓库: https://github.com/mritd/tpclash
     - Telegram: https://t.me/tpclash
`

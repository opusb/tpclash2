# Transparent proxy tool for Clash

> 这是一个用于 Clash Premium 的透明代理辅助工具, 由于众所周知周知的原因(**手笨**)而创建的.

## 一、TPClash 有啥用

> 说人话: 一个 TPClash 二进制文件 + 一个 Clash 配置文件 = 一条命令启动透明代理

TPClash 可以自动安装 Clash Premium, 并自动配置基于 TProxy 的透明代理; 透明代理同时支持 TCP 和 UDP 协议, 包括对 DNS 的自动配置和 ICMP 的劫持等.

**TPClash 的透明代理规则、日志配置、Dashboard(UI) 配置等全部从标准的 Clash Premium 配置文件内读取,并完成自适应; TPClash 暂时不会创建自己的自定义
配置文件(减轻使用负担).**

**同时 TPClash 在终止后会清理自己创建的 iptables 规则和路由表(防止把机器搞没); 这种清理不会简单的执行 `iptables -F/-X`, 而是进行 "定点清除", 以防止误删用户规则.**

## 二、TPClash 怎么用

TPClash 只有一个二进制文件, 直接从 Release 页面下载二进制文件运行即可. TPClash 二进制内嵌入了目标平台的 Clash 二进制文件以及其他资源文件(All in one), 
启动后会自动释放, 所以无需再下载 Clash. 

**注意: TPClash 默认会读取 位于 `/etc/clash.yaml` 的 clash 配置文件, 如果 clash 配置文件在其他位置请自行修改.**

```sh
./tpclash run -c /etc/clash.yaml
```

**初次使用的用户推荐命令行执行, 如果出现规则冲突导致断网情况(理论上不会)可以简单的通过重启解决. TPClash 支持的所有命令可以通过 `--help` 查看:**

```sh
root@test62 ~ # ❯❯❯ ./tpclash --help
Transparent proxy tool for Clash

Usage:
  tpclash [command]

Available Commands:
  run         Run tpclash
  clean       Clean tpclash iptables and route config
  extract     Extract embed files
  help        Help about any command
  completion  Generate the autocompletion script for the specified shell

Flags:
  -c, --config string   clash config path (default "/etc/clash.yaml")
  -h, --help            help for tpclash
  -d, --home string     clash home dir (default "/data/clash")
  -u, --ui string       clash dashboard(official/yacd) (default "yacd")

Use "tpclash [command] --help" for more information about a command.
```

## 三、TPClash 做了什么

**TPClash 在启动后会进行如下动作:**

- 1、创建 `/data/clash` 目录, 并将其作为 Clash 的 `Home Dir`
- 2、将 Clash Premium 二进制文件、Dashboard(官方+yacd)、必要的 ruleset、Country.mmdb 释放到 `/data/clash` 目录
- 3、创建 `tpclash` 普通用户用于启动 clash, 该用户用于配合 iptables 进行流量过滤
- 4、添加透明代理的路由表和 iptables 配置
- 5、启动官方的 Clash Premium, 并设置必要参数, 比如 `-ext-ui`、`-d` 等

## 四、如何编译 TPClash

由于 TPClash 是一个集成工具, 所以在编译前请安装好以下工具链:

- git
- curl
- jq
- tar
- gzip
- nodejs(用于编译 Dashboard)
- pnpm、yarn(Dashboard 编译所需依赖工具, 可通过 `npm i -g xxx` 安装)
- golang 1.17+
- [go-task](https://github.com/go-task/task)(类似 Makefile 的替代工具)

TPClash 项目内的 `Taskfile.yaml` 内已经写好了自动编译脚本, 只需要执行 `task` 命令即可:

```sh
git clone https://github.com/mritd/tpclash.git
cd tpclash
task # go-task 安装成功后会包含此命令
```

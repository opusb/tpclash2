# Transparent proxy tool for Clash

> 这是一个用于 Clash Premium 的透明代理辅助工具, 由于众所周知周知的原因(**手笨**)而创建的.

## 一、TPClash 是什么

TPClash 可以自动安装 Clash Premium, 并自动配置基于 Tun 的透明代理.

**TPClash 的透明代理规则、日志配置、Dashboard(UI) 配置等全部从标准的 Clash Premium 配置文件内读取, 并完成自适应; TPClash 暂时不会创建自己的自定义
配置文件(减轻使用负担).**

## 二、TPClash 使用

TPClash 只有一个二进制文件, 直接从 Release 页面下载二进制文件运行即可. TPClash 二进制内嵌入了目标平台的 Clash 二进制文件以及其他资源文件(All in one), 
启动后会自动释放, 所以无需再下载 Clash. 

**注意: TPClash 默认会读取位于 `/etc/clash.yaml` 的 clash 配置文件, 如果 clash 配置文件在其他位置请自行修改.**

> 从 v0.0.19 版本开始支持远程配置下载, 如果 `-c` 指定为 http 地址, 则 clash 在启动后将会自动下载远程配置到 "clash home" 中(`xclash.yaml`),
> 然后用该配置启动; 下载行为每次启动都会执行, 所以可以利用此功能和 [subconverter](https://github.com/tindy2013/subconverter) 实现自定订阅转换和定时刷新.

```sh
./tpclash -c /etc/clash.yaml
```

**由于 TPClash 只是一个辅助工具, 实际代理处理还是由 Clash 完成, 为了避免错误配置导致代理不工作, TPClash 对 Clash 配置文件进行了必要性的配置检测.**

### 2.1、TUN 模式配置

```yaml
# 请指定自己实际的接口名称(ip a 获取)
interface-name: ens160

# 需要开启 TUN 配置
tun:
  enable: true
  stack: system
  dns-hijack:
    - any:53
  #   - 8.8.8.8:53
  #   - tcp://8.8.8.8:53
  auto-route: true

# 开启 DNS 配置, 且使用 fake-ip 模式
dns:
  enable: true
  listen: 0.0.0.0:1053
  enhanced-mode: fake-ip
  default-nameserver:
    - 114.114.114.114
  nameserver:
    - 114.114.114.114
```

### 2.2、启动 TPClash

**初次使用的用户推荐命令行执行并增加 `--test` 参数, 该参数保证 TPClash 在启动 5 分钟后自动退出, 如果出现断网等情况也能自行恢复. TPClash 支持的所有命令可以通过 `--help` 查看:**

```sh
root@test62 ~ # ❯❯❯ ./tpclash --help
Transparent proxy tool for Clash

Usage:
  tpclash [flags]

Flags:
      --clash-user string   clash runtime user (default "tpclash")
  -c, --config string       clash config local path or remote url (default "/etc/clash.yaml")
      --debug               enable debug log
      --disable-extract     disable extract files
  -h, --help                help for tpclash
      --hijack-ip ipSlice   hijack target IP traffic (default [])
  -d, --home string         clash home dir (default "/data/clash")
      --test                run in test mode, exit automatically after 5 minutes
  -u, --ui string           clash dashboard(official|yacd) (default "yacd")
  -v, --version             version for tpclash
```

### 2.3、Meta 用户

从 `v0.0.16` 版本开始支持 Clash Meta 分支版本, Meta 用户**需要在配置文件中关闭 iptables 配置**:

```yaml
iptables:
  enable: false
```

### 2.4、设置流量转发

TPClash 启动成功后, 将其他主机的网关指向当前 TPClash 服务器 IP 即可实现透明代理;
对于其他主机请使用默认路由器 IP 或者类似 114 等公共 DNS 作为主机 DNS.
**请不要将其他主机的 DNS 也设置为 TPClash 服务器 IP, 因为当前 Clash 可能并未监听 53 端口.**

### 2.5、自动流量接管

从 `v0.0.13` 版本起, TPClash 内置了一个 ARP 流量劫持功能, 可以通过 `--hijack-ip` 选项指定需要劫持的 IP 地址:

```sh
# 可以指定多次
./tpclash --hijack-ip 172.16.11.92 --hijack-ip 172.16.11.93
```

当该选项被设置后, TPClash 将会对目标 IP 发起 ARP 攻击, 从而强制接管目标地址的流量. 需要注意的是, 当目标 IP 被设置为 `0.0.0.0`
时, TPClash 将会劫持所有内网流量, 这可能会因为配置错误导致整体断网, 所以请谨慎操作.

### 2.6、在Docker容器中使用

在 Docker 容器中使用需要创建 `/dev/net/tun` 设备并允许修改 `iptables`; 同时需要设置 `net.ipv4.ip_forward` 与 `net.ipv4.conf.all.route_localnet` 内核参数.

因此在创建容器时需要加入以下参数:

```sh
docker run \
  --sysctl net.ipv4.ip_forward=1 \
  --sysctl net.ipv4.conf.all.route_localnet=1 \
  --cap-add MKNOD \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  ubuntu:20.04
```

并在容器创建后创建 `/dev/net/tun` 设备:

```
mkdir /dev/net
mknod /dev/net/tun c 10 200
chmod 777 /dev/net/tun
```

之后便可正常在docker容器中使用 tpclash.

## 三、TPClash 做了什么

**TPClash 在启动后会进行如下动作:**

- 1、创建 `/data/clash` 目录, 并将其作为 Clash 的 `Home Dir`
- 2、将 Clash Premium 二进制文件、Dashboard(官方+yacd)、必要的 ruleset、Country.mmdb 释放到 `/data/clash` 目录
- 3、创建 `tpclash` 普通用户用于启动 clash
- 4、选择性添加透明代理的路由表和 iptables 配置
- 5、启动官方的 Clash Premium, 并设置必要参数, 比如 `-ext-ui`、`-d` 等

## 四、如何编译 TPClash

由于 TPClash 是一个集成工具, 所以在编译前请安装好以下工具链:

- git
- curl
- jq
- tar
- gzip
- nodejs(用于编译 Dashboard)
- pnpm(Dashboard 编译所需依赖工具, 可通过 `npm i -g xxx` 安装)
- golang 1.19+
- [go-task](https://github.com/go-task/task)(类似 Makefile 的替代工具)

TPClash 项目内的 `Taskfile.yaml` 内已经写好了自动编译脚本, 只需要执行 `task` 命令即可:

```sh
git clone https://github.com/mritd/tpclash.git
cd tpclash
task # go-task 安装成功后会包含此命令
```

## 五、其他说明

TPClash 默认释放的文件包含了 [Loyalsoldier/clash-rules](https://github.com/Loyalsoldier/clash-rules) 相关文件, 可在规则中直接使用;

**TPClash 同时也释放了 [Hackl0us/GeoIP2-CN](https://github.com/Hackl0us/GeoIP2-CN) 项目的 Country.mmdb 文件, 该 GeoIP 数据库
仅包含中国大陆地区 IP, 所以如果使用 `GEOIP,US,PROXY` 等其他国家规则会失败.**

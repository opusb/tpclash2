- [Transparent proxy tool for Clash](#transparent-proxy-tool-for-clash)
  - [一、TPClash 是什么](#一tpclash-是什么)
  - [二、TPClash 使用](#二tpclash-使用)
    - [2.1、TUN 模式配置](#21tun-模式配置)
    - [2.2、TUN 配合 eBPF 配置](#22tun-配合-ebpf-配置)
    - [2.3、启动 TPClash](#23启动-tpclash)
    - [2.4、Meta 用户](#24meta-用户)
    - [2.5、设置流量转发](#25设置流量转发)
    - [2.6、在 Docker 容器中使用](#26在-docker-容器中使用)
    - [2.7、远程配置加载](#27远程配置加载)
    - [2.8、使用加密的配置文件](#28使用加密的配置文件)
    - [2.9、容器化虚拟机部署](#29容器化虚拟机部署)
  - [三、TPClash 做了什么](#三tpclash-做了什么)
  - [四、如何编译 TPClash](#四如何编译-tpclash)
  - [五、其他说明](#五其他说明)
  - [六、官方讨论群](#六官方讨论群)

# Transparent proxy tool for Clash

> 这是一个用于 Clash Premium 的透明代理辅助工具, 由于众所周知周知的原因(**手笨**)而创建的.

## 一、TPClash 是什么

TPClash 可以自动安装 Clash Premium/Meta, 并自动配置基于 Tun 的透明代理.

**TPClash 的透明代理规则、日志配置、Dashboard(UI) 配置等全部从标准的 Clash 配置文件内读取, 并完成自适应; TPClash 暂时不会创建自己的自定义
配置文件(减轻使用负担).**

## 二、TPClash 使用

TPClash 只有一个二进制文件, 直接从 Release 页面下载二进制文件运行即可. TPClash 二进制内嵌入了目标平台的 Clash 二进制文件以及其他资源文件(All in one), 
启动后会自动释放, 所以无需再下载 Clash. 

**注意: TPClash 默认会读取位于 `/etc/clash.yaml` 的 clash 配置文件, 如果 clash 配置文件在其他位置请自行修改.**

```sh
./tpclash -c /etc/clash.yaml
```

TPClash 会自动监视配置文件变动并自动完成重载, 同时可以在配置文件中使用 `{{IfName}}` 代替本机网卡名称, TPClash 会自动检测并进行替换;
具体配置可参考项目下的 [example.yaml](https://github.com/mritd/tpclash/blob/master/example.yaml).

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
  fake-ip-range: 198.18.0.1/16
  default-nameserver:
    - 223.5.5.5
    - 119.29.29.29
  nameserver:
    - 223.5.5.5
    - 119.29.29.29
```

### 2.2、TUN 配合 eBPF 配置

```yaml
# 请指定自己实际的接口名称(ip a 获取)
interface-name: ens160

tun:
  enable: true
  stack: system
  dns-hijack:
    - any:53
  #   - 8.8.8.8:53
  #   - tcp://8.8.8.8:53
  # auto-route 与 ebpf 冲突, 不能同时使用
  #auto-route: true

# ebpf 需要指定物理网卡
ebpf:
  redirect-to-tun:
    - ens34

# ebpf 需要配置 mark
routing-mark: 666

# 开启 DNS 配置, 且使用 fake-ip 模式
dns:
  enable: true
  listen: 0.0.0.0:1053
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  default-nameserver:
    - 223.5.5.5
    - 119.29.29.29
  nameserver:
    - 223.5.5.5
    - 119.29.29.29
```

### 2.3、启动 TPClash

**TPClash 支持的所有命令可以通过 `--help` 查看:**

```sh
root@test51 ~ # ❯❯❯ ./tpclash --help
Transparent proxy tool for Clash

Usage:
  tpclash [flags]

Flags:
  -i, --check-interval duration   remote config check interval (default 2m0s)
  -c, --config string             clash config local path or remote url (default "/etc/clash.yaml")
      --debug                     enable debug log
      --disable-extract           disable extract files
  -h, --help                      help for tpclash
  -d, --home string               clash home dir (default "/data/clash")
      --http-header strings       http header when requesting a remote config(key=value)
  -u, --ui string                 clash dashboard(official|yacd) (default "yacd")
  -v, --version                   version for tpclash
```

### 2.4、Meta 用户

> 注意: Meta 版本暂时没有经过严格的测试, 作者并没有使用 Meta 版本的需求.

从 `v0.0.16` 版本开始支持 Clash Meta 分支版本, Meta 用户**需要在配置文件中关闭 iptables 配置**:

```yaml
iptables:
  enable: false
```

### 2.5、设置流量转发

TPClash 启动成功后, 将其他主机的网关指向当前 TPClash 服务器 IP 即可实现透明代理; 对于被代理主机请使用公网 DNS.

**请不要将其他主机的 DNS 也设置为 TPClash 服务器 IP, 因为这回导致一些不可预测的问题, 具体请参考 [Clash DNS 科普](https://github.com/mritd/tpclash/wiki/Clash-DNS-%E7%A7%91%E6%99%AE).**

### 2.6、在 Docker 容器中使用

> 注意: 从 `v0.1.0` 版本开始, 如果使用 Docker 运行或者宿主机安装了 Docker, **TPClash 会自动尝试使用 nftables 进行修复;**
> 如果宿主机不支持 nftables, 请自行使用 `iptables -I DOCKER-USER -i src_if -o dst_if -j ACCEPT` 命令修复.

如果想要在 Docker 中使用 tpclash, 只需要挂载外部配置文件即可:

```sh
docker run -dt \
  --name tpclash \
  --privileged \
  --network=host \
  -v /root/clash.yaml:/etc/clash.yaml \
  mritd/tpclash
```

**此命令假设配置文件位于宿主机的 `/root/clash.yaml` 位置, 其他位置请自行替换; 该镜像采用 [Earthly](https://earthly.dev/) 编译, Earthfile 存储在 [autobuild](https://github.com/mritd/autobuild/tree/main/tpclash) 仓库.**

### 2.7、远程配置加载

为了方便使用, 在 `v0.0.19` 版本开始支持远程配置加载; 从 `v0.0.22` 版本开始进一步优化远程配置加载功能, 目前使用方式如下:

- 1、使用 `-c` 参数指定 http(s) 远程配置文件地址, 例如 `-c http://127.0.0.1:8080/clash.yaml`
- 2、使用 `-i` 参数指定检查间隔时间, tpclash 会按照这个时间频率去检查远程配置是否与本地一致, 不一致则更新并自动重载
- 3、增加了 `--http-header` 选项用于用于设置下载远程配置的 http 请求头, 用于支持下载公网带认证的托管配置, 例如 `--http-header "Authorization=Basic YWRtaW46MTIz"`

**注意: 如果远程配置修改了端口等配置, 那么仍需要重新启动 TPClash, 因为 TPClash 重载无法照顾到底层的端口变更.**

### 2.8、使用加密的配置文件

从 `v0.1.6` 版本开始支持配置文件加密, 现在可以使用以下命令对明文的 yaml 配置进行加密:

```sh
./tpclash enc --config-password YOUR_PASSWORD clash.yaml
```

加密完成后将在本地生成一个被加密过的 `clash.yaml.enc` 文件, 该文件可以直接托管到任何可公共访问的 http 地址(也可以本地使用).


**当 TPClash 指定了远程 http 配置, 同时 `--config-password` 选项不为空的情况下, 则认为远程地址的配置文件是被加密的,
TPClash 将会自动完成解密并加载:**

```sh
./tpclash --config-password YOUR_PASSWORD -c https://exmaple.com/clash.yaml.enc
```

### 2.9、容器化虚拟机部署

TPClash 一开始的目标就是作为一个稳定可靠的、可以直接托管配置的内网网关使用, 虽然 TPClash 可以兼容大多数系统, 但特殊系统环境例如 OpenWrt 等可能会出现一些兼容性问题;
为了统一部署环境和更方便使用, 目前已增加了纯容器化系统 Fedora CoreOS 和 Flatcar 系统支持, 这两个系统都支持直接使用单个配置文件完成引导和自动化部署, 且后台自动滚动升级;
可以持续维持系统的最新状态并且可安全回滚; 以下为两个系统在 ESXi 下的直接部署说明:

- 1、下载项目内的 fedora-coreos.butane.yaml 或 flatcar.butane.yaml
- 2、调整配置文件内的 IP 地址和网关地址, DNS 一般不需要修改
- 3、调整配置文件内 TPClash 的启动命令, 一般需要指定远程 clash 配置文件地址
- 4、参考 [butane](https://coreos.github.io/butane/getting-started/) 官方文档安装 butane 工具
- 5、执行 `butane --pretty --strict flatcar.butane.yaml | base64 -w0` 生成 base64 编码格式的 Ignition 配置(yaml名称自行替换)
- 6、下载对应系统的 ova 系统部署文件, Fedora CoreOS [点击这里](https://fedoraproject.org/coreos/download/?stream=stable#baremetal), Flatcar [点击这里](https://stable.release.flatcar-linux.net/amd64-usr/current/flatcar_production_vmware_ova.ova)
- 7、在 ESXi 内创建虚拟机选择从 `OVA` 部署
- 8、Fedora CoreOS 部署时 **其他设置/Options** 中 `Ignition config data` 填写第 5 步生成的 base64 字符串, `Ignition config data encoding` 填写 `base64`
- 9、Flatcar 部署时 **其他设置/Options** 中 `Ignition/coreos-cloudinit data` 填写第 5 步生成的 base64 字符串, `Ignition/coreos-cloudinit data encoding` 填写 `base64`
- 10、最后启动完成, 一个容器化、不可变的可靠 TPClash 网关就启动了

关于这两个系统以及其配置文件限于篇幅无法做过多说明, 推荐阅读以下官方文档, 如有其他需要帮助或疑问请开 issue.


## 三、TPClash 做了什么

**TPClash 在启动后会进行如下动作:**

- 1、创建 `/data/clash` 目录(可自行指定成其他目录), 并将其作为 Clash 的 `Home Dir`
- 2、将 Clash 二进制文件、Dashboard(官方+yacd)、必要的 ruleset、Country.mmdb 释放到 `/data/clash` 目录
- 3、从本地或远程读取配置, 进行模版解析后复制到 `/data/clash/xclash.yaml`
- 4、启动官方的 Clash, 并设置必要参数, 比如 `-ext-ui`、`-d` 等
- 5、选择性进行网络配置, 例如为 Docker 用户自动设置 nftables
- 6、在后台持续监视本地或远程配置文件变动, 然后自动重载

## 四、如何编译 TPClash

由于 TPClash 是一个集成工具, 所以在编译前请安装好以下工具链:

- git
- curl
- jq
- tar
- gzip
- nodejs(用于编译 Dashboard)
- pnpm(Dashboard 编译所需依赖工具, 可通过 `npm i -g xxx` 安装)
- golang 1.20+
- [go-task](https://github.com/go-task/task)(类似 Makefile 的替代工具)

TPClash 项目内的 `Taskfile.yaml` 内已经写好了自动编译脚本, 只需要执行 `task` 命令即可:

```sh
git clone https://github.com/mritd/tpclash.git
cd tpclash
task # go-task 安装成功后会包含此命令
```

**其他高级编译(例如单独编译特定平台)请执行 `task --list` 查看.**

## 五、其他说明

TPClash 默认释放的文件包含了 [Loyalsoldier/clash-rules](https://github.com/Loyalsoldier/clash-rules) 相关文件, 可在规则中直接使用;

**TPClash 同时也释放了 [Hackl0us/GeoIP2-CN](https://github.com/Hackl0us/GeoIP2-CN) 项目的 Country.mmdb 文件, 该 GeoIP 数据库
仅包含中国大陆地区 IP, 所以如果使用 `GEOIP,US,PROXY` 等其他国家规则会失败.**

## 六、官方讨论群

Telegram: [https://t.me/tpclash](https://t.me/tpclash)

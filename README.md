# Transparent proxy tool for Clash

> 这是一个用于 Clash Premium 的透明代理辅助工具, 由于众所周知周知的原因(**手笨**)而创建的.

## 一、TPClash 是什么

TPClash 可以自动安装 Clash Premium/Meta, 并自动配置基于 Tun 的透明代理.

**TPClash 的透明代理规则、日志配置、Dashboard(UI) 配置等全部从标准的 Clash 配置文件内读取, 并完成自适应; TPClash 暂时不会创建自己的自定义
配置文件(减轻使用负担).**

## 二、TPClash 使用

### 2.1、直接启动

TPClash 只有一个二进制文件, 直接从 Release 页面下载二进制文件运行即可. TPClash 二进制内嵌入了目标平台的 Clash 二进制文件以及其他资源文件(All in one), 
启动后会自动释放, 所以无需再下载 Clash. 

**注意: TPClash 默认会读取位于 `/etc/clash.yaml` 的 clash 配置文件, 如果 clash 配置文件在其他位置请自行修改.**

```sh
./tpclash-premium-linux-amd64-v3 -c /etc/clash.yaml
```

### 2.2、Systemd 安装

除了直接运行之外, 针对于支持 Systemd 的系统 TPClash 也支持 install 命令用于将自身安装为 Systemd 服务; **安装时 TPClash 先将自身复制
到 `/usr/local/bin/tpclash`, 然后创建 `/etc/systemd/system/tpclash.service` 配置文件, 并且将附加参数也同步写入到 Systemd 配置中.**

```sh
root@tpclash ~ # ❯❯❯ ./tpclash-premium-linux-amd64-v3 install --config https://example.com/clash.yaml

████████╗██████╗  ██████╗██╗      █████╗ ███████╗██╗  ██╗
╚══██╔══╝██╔══██╗██╔════╝██║     ██╔══██╗██╔════╝██║  ██║
   ██║   ██████╔╝██║     ██║     ███████║███████╗███████║
   ██║   ██╔═══╝ ██║     ██║     ██╔══██║╚════██║██╔══██║
   ██║   ██║     ╚██████╗███████╗██║  ██║███████║██║  ██║
   ╚═╝   ╚═╝      ╚═════╝╚══════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
  👌 TPClash 安装完成, 您可以使用以下命令启动:
     - 启动服务: systemctl start tpclash
     - 停止服务: systemctl stop tpclash
     - 重启服务: systemctl restart tpclash
     - 开启自启动: systemctl enable tpclash
     - 关闭自启动: systemctl disable tpclash
     - 查看日志: journalctl -fu tpclash
     - 重载服务配置: systemctl daemon-reload
```

### 2.3、Docker 运行

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

### 2.4、容器化虚拟机部署

TPClash 一开始的目标就是作为一个稳定可靠的、可以直接托管配置的内网网关使用, 虽然 TPClash 可以兼容大多数系统, 但特殊系统环境例如 OpenWrt 等可能会出现
一些兼容性问题;为了统一部署环境和更方便使用, 目前已增加了纯容器化系统 Flatcar 系统支持, 该系统支持直接使用单个配置文件完成引导和自动化部署, 且后台自动
滚动升级; 可以持续维持系统的最新状态并且可安全回滚; 以下是在 ESXi 中直接部署说明:

- 1、下载项目内的 flatcar.butane.yaml
- 2、调整配置文件内的 IP 地址和网关地址, DNS 一般不需要修改
- 3、调整配置文件内 TPClash 的启动命令, 一般需要指定远程 clash 配置文件地址
- 4、参考 [butane](https://coreos.github.io/butane/getting-started/) 官方文档安装 butane 工具
- 5、执行 `butane --pretty --strict flatcar.butane.yaml | base64 -w0` 生成 base64 编码格式的 Ignition 配置(yaml名称自行替换)
- 6、下载对应系统的 ova [系统部署文件](https://stable.release.flatcar-linux.net/amd64-usr/current/flatcar_production_vmware_ova.ova)
- 7、在 ESXi 内创建虚拟机选择从 `OVA` 部署
- 8、部署时 **其他设置/Options** 中 `Ignition/coreos-cloudinit data` 填写第 5 步生成的 base64 字符串, `Ignition/coreos-cloudinit data encoding` 填写 `base64`
- 9、最后启动完成, 一个容器化、不可变的可靠 TPClash 网关就启动了

关于这个系统以及其配置文件限于篇幅无法做过多说明, 推荐阅读[博客文章](https://mritd.com/2023/07/20/containerized-system-test/).

### 2.5、设置流量转发

TPClash 启动成功后, 将其他主机的网关指向当前 TPClash 服务器 IP 即可实现透明代理; 对于被代理主机请使用公网 DNS.

**请不要将其他主机的 DNS 也设置为 TPClash 服务器 IP, 因为这可能导致一些不可预测的问题, 具体请参考 [Clash DNS 科普](https://github.com/mritd/tpclash/wiki/Clash-DNS-%E7%A7%91%E6%99%AE).**

### 2.6、升级 TPClash

对于二进制文件部署的用户, 可以使用以下命令升级到最新版本:

```bash
root@tpclash ~ # ❯❯❯ tpclash upgrade
```

如果想要升级到特定版本也可以指定版本号:

```bash
root@tpclash ~ # ❯❯❯ tpclash upgrade v0.1.10
```

**升级前请确保关闭了 tpclash 服务, 升级时默认使用 `https://ghproxy.com` 进行加速, 如果不想使用可以通过 `--with-ghproxy=false` 选项关闭.**

## 三、TPClash 配置

默认情况下 TPClash 会读取 `/etc/clash.yaml` 配置文件启动 Clash; **TPClash 首先会读取该文件并进行模版解析, 解析成功后 TPClash 会将其写入到 Home 目录的 `xclash.yaml` 中
(默认为 `/data/clash/xclash.yaml`), 然后再使用该配置启动 Clash.** 由于 TPClash 只是一个辅助工具, 实际代理处理还是由 Clash 完成, 为了避免错误配置导致代理不工作, TPClash
对 Clash 配置文件进行了必要性的配置检测. 下面是一些推荐的配置样例:

### 3.1、TUN 模式配置

```yaml
# 需要开启 TUN 配置
tun:
  enable: true
  stack: system
  dns-hijack:
    - any:53
  #   - 8.8.8.8:53
  #   - tcp://8.8.8.8:53
  auto-route: true
  auto-redir: true
  auto-detect-interface: true

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

### 3.2、TUN 配合 eBPF 配置

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
  #auto-redir: true
  #auto-detect-interface: true

# ebpf 需要指定物理网卡
ebpf:
  redirect-to-tun:
    - ens160

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

### 3.3、Meta 用户

> 注意: Meta 版本暂时没有经过严格的测试, 作者并没有使用 Meta 版本的需求.

从 `v0.0.16` 版本开始支持 Clash Meta 分支版本, Meta 用户**需要在配置文件中关闭 iptables 配置**, 其他配置与默认的 Permium 版本相同:

```yaml
iptables:
  enable: false
```

### 3.4、订阅用户

如果期望完全不修改订阅配置实现透明代理, 可直接使用 `--auto-fix=tun` 参数启动, **该参数将会自动修补远程配置来实现透明代理, 同样带来的
后果是一些参数将会被硬编码:**

```sh
root@tpclash ~ # ❯❯❯ tpclash --auto-fix tun -c https://exmaple.com/clash.yaml
```

## 四、高级配置

### 4.1、远程配置加载

为了方便使用, 在 `v0.0.19` 版本开始支持远程配置加载; 从 `v0.0.22` 版本开始进一步优化远程配置加载功能, 目前使用方式如下:

- 1、使用 `-c` 参数指定 http(s) 远程配置文件地址, 例如 `-c https://example.com/clash.yaml`
- 2、使用 `-i` 参数指定检查间隔时间, TPClash 会按照这个时间频率去检查远程配置是否与本地一致, 不一致则更新并自动重载
- 3、使用 `--http-header` 参数设置下载远程配置的 http 请求头, 用于支持下载公网带认证的托管配置, 例如 `--http-header "Authorization=Basic YWRtaW46MTIz"`
- 4、使用 `--config-password` 参数设置配置文件的密码, 改密码用于解密配置文件, 主要用于将配置文件存储在可公共访问的地址(防止泄密)

**注意: 如果远程配置修改了端口等配置, 那么仍需要重新启动 TPClash, 因为 TPClash 重载无法照顾到底层的端口变更.**

### 4.2、使用加密的配置文件

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

### 4.3、使用模版引擎

为了应对单配置文件多实例的部署情况, TPClash 内置了一些模版函数, 这些函数可以辅助配置生成完成自动化配置:

- `{{IfName}}`: 自动解析为当前主机的主网卡
- `{{DefaultDNS}}`: 自动获取当前主机默认的上游 DNS

模版函数可能随后续更新继续添加, 使用方法请参考项目内的 [example.yaml](https://github.com/mritd/tpclash/blob/master/example.yaml) 配置.

### 4.4、Premium Tracing

从 `v0.1.8` 版本开始提供 Premium 核心的 [Tracing Dashboard](https://github.com/Dreamacro/clash-tracing) 自动部署, **此功能需要宿主机安装有 Docker, TPClash 会调用 Docker API 自动创建容器.**

对于采用 Systemd 部署的用户, 宿主机安装好 Docker 后无需其他特殊操作; 对于采用 Docker 部署的用户, 需要增加一个挂载:

```diff
docker run -dt \
  --name tpclash \
  --privileged \
  --network=host \
  -v /root/clash.yaml:/etc/clash.yaml \
+ -v /var/run/docker.sock:/var/run/docker.sock \
  mritd/tpclash
```

**然后需要在配置文件中开启 Tracing:**

```yaml
profile:
    tracing: true
```

**最后启动 TPClash 时增加 Tracing 选项即可:**

```sh
./tpclash-premium-linux-amd64-v3 --enable-tracing -c /etc/clash.yaml
```

启动完成后可访问 `http://TPCLASH_IP:3000` 查看 Tracing Dashboard, 其默认账户密码均为 `admin`.

## 五、TPClash 做了什么

**TPClash 在启动后会进行如下动作:**

- 1、创建 `/data/clash` 目录(可自行指定成其他目录), 并将其作为 Clash 的 `Home Dir`
- 2、将 Clash 二进制文件、Dashboard(官方+yacd)、必要的 ruleset、Country.mmdb 释放到 `/data/clash` 目录
- 3、从本地或远程读取配置, 进行模版解析后复制到 `/data/clash/xclash.yaml`
- 4、启动官方的 Clash, 并设置必要参数, 比如 `-ext-ui`、`-d` 等
- 5、选择性进行网络配置, 例如为 Docker 用户自动设置 nftables
- 6、在后台持续监视本地或远程配置文件变动, 然后自动重载

## 六、如何编译 TPClash

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

## 七、其他说明

TPClash 默认释放的文件包含了 [Loyalsoldier/clash-rules](https://github.com/Loyalsoldier/clash-rules) 相关文件, 可在规则中直接使用;

**TPClash 同时也释放了 [Hackl0us/GeoIP2-CN](https://github.com/Hackl0us/GeoIP2-CN) 项目的 Country.mmdb 文件, 该 GeoIP 数据库
仅包含中国大陆地区 IP, 所以如果使用 `GEOIP,US,PROXY` 等其他国家规则会失败.**

## 八、官方讨论群

Telegram: [https://t.me/tpclash](https://t.me/tpclash)

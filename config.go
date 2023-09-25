package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"golang.org/x/crypto/chacha20poly1305"

	"gopkg.in/yaml.v3"

	"github.com/fsnotify/fsnotify"

	"github.com/sirupsen/logrus"
)

type TPClashConf struct {
	ClashHome         string
	ClashConfig       string
	ClashUI           string
	HttpHeader        []string
	HttpTimeout       time.Duration
	CheckInterval     time.Duration
	ConfigEncPassword string
	AutoFixMode       string

	ForceExtract         bool
	EnableTracing        bool
	PrintVersion         bool
	UpgradeWithGhProxy   bool
	AllowStandardDNSPort bool

	Test  bool
	Debug bool
}

type ClashConf struct {
	Port               int    `yaml:"port"`
	SocksPort          int    `yaml:"socks-port"`
	MixedPort          int    `yaml:"mixed-port"`
	AllowLan           bool   `yaml:"allow-lan"`
	BindAddress        string `yaml:"bind-address"`
	Mode               string `yaml:"mode"`
	LogLevel           string `yaml:"log-level"`
	Ipv6               bool   `yaml:"ipv6"`
	ExternalController string `yaml:"external-controller"`
	ExternalUI         string `yaml:"external-ui"`
	Secret             string `yaml:"secret"`
	InterfaceName      string `yaml:"interface-name"`
	Ebpf               struct {
		RedirectToTun []string `yaml:"redirect-to-tun"`
	} `yaml:"ebpf"`
	RoutingMark int `yaml:"routing-mark"`
	Tun         struct {
		Enable              bool     `yaml:"enable"`
		Stack               string   `yaml:"stack"`
		DNSHijack           []string `yaml:"dns-hijack"`
		AutoRedir           bool     `yaml:"auto-redir"`
		AutoRoute           bool     `yaml:"auto-route"`
		AutoDetectInterface bool     `yaml:"auto-detect-interface"`
	} `yaml:"tun"`
	DNS struct {
		Enable            bool     `yaml:"enable"`
		Listen            string   `yaml:"listen"`
		Ipv6              bool     `yaml:"ipv6"`
		DefaultNameserver []string `yaml:"default-nameserver"`
		EnhancedMode      string   `yaml:"enhanced-mode"`
		FakeIPRange       string   `yaml:"fake-ip-range"`
		FakeIPFilter      []string `yaml:"fake-ip-filter"`
		Nameserver        []string `yaml:"nameserver"`
	} `yaml:"dns"`

	// Meta
	IPTables struct {
		Enable bool `yaml:"enable"`
	} `yaml:"iptables"`
}

func CheckConfig(c string) (*ClashConf, error) {
	var cc ClashConf
	if err := yaml.Unmarshal([]byte(c), &cc); err != nil {
		return nil, fmt.Errorf("[config] failed to unmarshal clash config: %w", err)
	}

	// common check
	if strings.ToLower(cc.DNS.EnhancedMode) != "fake-ip" {
		return nil, fmt.Errorf("[config] only support fake-ip dns mode(dns.enhanced-mode)")
	}

	dnsHost, dnsPort, err := net.SplitHostPort(cc.DNS.Listen)
	if err != nil {
		return nil, fmt.Errorf("[config] failed to parse clash dns listen config(dns.listen): %w", err)
	}

	dport, err := strconv.Atoi(dnsPort)
	if err != nil {
		return nil, fmt.Errorf("[config] failed to parse clash dns listen config(dns.listen): %w", err)
	}
	if dport < 1 {
		return nil, fmt.Errorf("[config] dns port in clash config is missing(dns.listen)")
	}
	if !conf.AllowStandardDNSPort && dport == 53 {
		return nil, fmt.Errorf("[config] please do not set DNS to listen on port 53(dns.listen), see also: https://github.com/mritd/tpclash/wiki/Clash-DNS-%E7%A7%91%E6%99%AE")
	}

	dhost := net.ParseIP(dnsHost)
	if dhost == nil {
		return nil, fmt.Errorf("[config] dns listening address parse failed(dns.listen): is not a valid IP address")
	}

	if cc.InterfaceName == "" && !cc.Tun.AutoDetectInterface {
		return nil, fmt.Errorf("[config] failed to parse clash interface name(interface-name): interface-name or tun.auto-detect-interface must be set")
	}

	if cc.DNS.FakeIPRange == "" {
		return nil, fmt.Errorf("[config] failed to parse clash fake ip range name(dns.fake-ip-range): fake-ip-range must be set")
	}

	if !cc.Tun.Enable {
		return nil, fmt.Errorf("[config] tun must be enabled in tun mode(tun.enable)")
	}

	if !cc.Tun.AutoRoute && len(cc.Ebpf.RedirectToTun) == 0 {
		return nil, fmt.Errorf("[config] must be enabled auto-route or ebpf in tun mode(tun.auto-route/ebpf.redirect-to-tun)")
	}

	if cc.Tun.AutoRoute && len(cc.Ebpf.RedirectToTun) > 0 {
		return nil, fmt.Errorf("[config] cannot enable auto-route and ebpf at the same time(tun.auto-route/ebpf.redirect-to-tun)")
	}

	if cc.RoutingMark == 0 && len(cc.Ebpf.RedirectToTun) > 0 {
		return nil, fmt.Errorf("[config] ebpf needs to set routing-mark(routing-mark)")
	}

	if cc.IPTables.Enable {
		return nil, fmt.Errorf("[config] meta kernel must turn off iptables(iptables.enable)")
	}

	return &cc, nil
}

func WatchConfig(ctx context.Context) chan string {
	buffer := ""
	updateCh := make(chan string, 3)

	if strings.HasPrefix(conf.ClashConfig, "http://") || strings.HasPrefix(conf.ClashConfig, "https://") {
		ccStr, err := loadRemoteConfig()
		if err != nil {
			logrus.Fatal(err)
		}
		buffer = ccStr
		updateCh <- autoFix(ccStr)

		go func() {
			tick := time.Tick(conf.CheckInterval)
			for {
				select {
				case <-ctx.Done():
					close(updateCh)
					logrus.Warnf("[config] stop config watching...")
					return
				case <-tick:
					ccStr, err = loadRemoteConfig()
					if err != nil {
						logrus.Error(err)
						continue
					}
					if ccStr != buffer {
						buffer = ccStr
						updateCh <- autoFix(ccStr)
					}
				}
			}
		}()
	} else {
		ccStr, err := loadLocalConfig()
		if err != nil {
			logrus.Fatal(err)
		}
		buffer = ccStr
		updateCh <- autoFix(ccStr)

		go func() {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				logrus.Fatalf("[config] failed to create fs watcher: %v", err)
			}
			defer func() { _ = watcher.Close() }()

			if err = watcher.Add(filepath.Dir(conf.ClashConfig)); err != nil {
				logrus.Fatalf("[config] failed add %s to fs watcher: %v", conf.ClashConfig, err)
			}

			for {
				select {
				case <-ctx.Done():
					close(updateCh)
					logrus.Warnf("[config] stop config watching...")
					return
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Name != conf.ClashConfig {
						continue
					}
					if event.Has(fsnotify.Write) {
						ccStr, err = loadLocalConfig()
						if err != nil {
							logrus.Error(err)
							continue
						}
						if ccStr != buffer {
							buffer = ccStr
							updateCh <- autoFix(ccStr)
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					if err != nil {
						logrus.Errorf("[config] fs watcher error: %v", err)
					}
				}
			}
		}()
	}
	return updateCh
}

func AutoReload(updateCh chan string, writePath string) {
	for ccStr := range updateCh {
		logrus.Info("[config] clash config changed, reloading...")

		ccStr = autoFix(ccStr)
		cc, err := CheckConfig(ccStr)
		if err != nil {
			logrus.Errorf("[config] an error was detected in the clash config, skipping automatic reload:\n %v", err)
			continue
		}

		if err := os.WriteFile(writePath, []byte(ccStr), 0644); err != nil {
			logrus.Errorf("[config] failed to copy clash config: %v", err)
			continue
		}

		apiAddr := cc.ExternalController
		if apiAddr == "" {
			apiAddr = "127.0.0.1:9090"
		}
		secret := cc.Secret

		req, err := http.NewRequest("PUT", "http://"+apiAddr+"/configs", bytes.NewReader([]byte(fmt.Sprintf(`{"path": "%s"}`, writePath))))
		if err != nil {
			logrus.Errorf("[config] failed to create reload req: %v", err)
			continue
		}
		req.Header.Set("Authorization", "Bearer "+secret)
		cli := &http.Client{Timeout: 5 * time.Second}

		resp, err := cli.Do(req)
		if err != nil {
			logrus.Errorf("[config] failed to reload config: %v", err)
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
			var msg bytes.Buffer
			_, _ = io.Copy(&msg, resp.Body)
			logrus.Errorf("[config] failed to reload config: status %d: %s", resp.StatusCode, msg.String())
		}

		logrus.Info("[config] clash config reload success...")
	}
}

func Encrypt(plaintext []byte, password string) []byte {
	key := sha256.Sum256([]byte(password))
	aead, _ := chacha20poly1305.NewX(key[:])

	return aead.Seal(nil, make([]byte, aead.NonceSize()), plaintext, nil)
}

func Decrypt(ciphertext []byte, password string) ([]byte, error) {
	key := sha256.Sum256([]byte(password))
	aead, _ := chacha20poly1305.NewX(key[:])

	return aead.Open(nil, make([]byte, aead.NonceSize()), ciphertext, nil)
}

func tplRendering(c string) string {
	var buf bytes.Buffer

	tpl, err := template.New("").Funcs(confFuncsMap).Parse(c)
	if err != nil {
		logrus.Errorf("[tplRendering] failed to parse template: %v", err)
		return c
	}

	// Auto-inject some value
	if err = tpl.Execute(&buf, nil); err != nil {
		logrus.Errorf("[tplRendering] failed to execute template: %v", err)
		return c
	}

	return buf.String()
}

func loadRemoteConfig() (string, error) {
	logrus.Debugf("[config] checking remote config...")

	req, err := http.NewRequest("GET", conf.ClashConfig, nil)
	if err != nil {
		return "", fmt.Errorf("[config] failed to create remote config req: %w", err)
	}

	for _, kv := range conf.HttpHeader {
		ss := strings.Split(kv, "=")
		if len(ss) != 2 {
			return "", fmt.Errorf("[config] failed to parse http header: %s", kv)
		}
		req.Header.Set(ss[0], ss[1])
	}

	req.Header.Set("User-Agent", fmt.Sprintf("TPClash %s %s", version, commit))

	cli := &http.Client{Timeout: conf.HttpTimeout}
	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("[config] failed to download remote config: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return "", fmt.Errorf("[config] failed to get remote config: status code %d", resp.StatusCode)
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("[config] failed to copy resp: %w", err)
	}

	if conf.ConfigEncPassword != "" {
		plaintext, err := Decrypt(bs, conf.ConfigEncPassword)
		return string(plaintext), err
	}

	return string(bs), nil
}

func loadLocalConfig() (string, error) {
	logrus.Debugf("[config] checking local config...")

	bs, err := os.ReadFile(conf.ClashConfig)
	if err != nil {
		return "", fmt.Errorf("[config] local config read error: %w", err)
	}

	if conf.ConfigEncPassword != "" {
		plaintext, err := Decrypt(bs, conf.ConfigEncPassword)
		return string(plaintext), err
	}

	return string(bs), nil
}

func autoFix(c string) string {
	c = tplRendering(c)

	if conf.AutoFixMode == "" {
		return c
	}

	logrus.Infof("[autofix] enable config auto fix...")

	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(c), &rootNode); err != nil {
		logrus.Errorf("[autofix] failed to unmarshal yaml config: %v", err)
		return c
	}

	var bindAddressNode yaml.Node
	_ = yaml.Unmarshal([]byte(tplRendering(bindAddressPatch)), &bindAddressNode)
	if !setYamlNode(&rootNode, "bind-address", bindAddressNode.Content[0]) {
		logrus.Error("[autofix] failed to patch bind-address config")
		return c
	}

	var externalControllerNode yaml.Node
	_ = yaml.Unmarshal([]byte(tplRendering(externalControllerPatch)), &externalControllerNode)
	if !setYamlNode(&rootNode, "external-controller", externalControllerNode.Content[0]) {
		logrus.Error("[autofix] failed to patch external-controller config")
		return c
	}

	var secretNode yaml.Node
	_ = yaml.Unmarshal([]byte(tplRendering(secretPatch)), &secretNode)
	if !setYamlNode(&rootNode, "secret", secretNode.Content[0]) {
		logrus.Error("[autofix] failed to patch secret config")
		return c
	}

	var nicNode yaml.Node
	_ = yaml.Unmarshal([]byte(tplRendering(nicPatch)), &nicNode)
	if !setYamlNode(&rootNode, "interface-name", nicNode.Content[0]) {
		logrus.Error("[autofix] failed to patch nic config")
		return c
	}

	var dnsNode yaml.Node
	_ = yaml.Unmarshal([]byte(tplRendering(dnsPatch)), &dnsNode)
	if !setYamlNode(&rootNode, "dns", dnsNode.Content[0]) {
		logrus.Error("[autofix] failed to patch dns config")
		return c
	}

	if conf.AutoFixMode == "ebpf" {
		var tunNode yaml.Node
		_ = yaml.Unmarshal([]byte(tplRendering(tunEBPFPatch)), &tunNode)
		if !setYamlNode(&rootNode, "tun", tunNode.Content[0]) {
			logrus.Error("[autofix] failed to patch tun config")
			return c
		}

		var ebpfNode yaml.Node
		_ = yaml.Unmarshal([]byte(tplRendering(ebpfPatch)), &ebpfNode)
		if !setYamlNode(&rootNode, "ebpf", ebpfNode.Content[0]) {
			logrus.Error("[autofix] failed to patch ebpf config")
			return c
		}

		var routingMarkNode yaml.Node
		_ = yaml.Unmarshal([]byte(tplRendering(routingMarkPatch)), &routingMarkNode)
		if !setYamlNode(&rootNode, "routing-mark", routingMarkNode.Content[0]) {
			logrus.Error("[autofix] failed to patch routing-mark config")
			return c
		}
	} else {
		var tunNode yaml.Node
		_ = yaml.Unmarshal([]byte(tplRendering(tunStandardPatch)), &tunNode)
		if !setYamlNode(&rootNode, "tun", tunNode.Content[0]) {
			logrus.Error("[autofix] failed to patch tun config")
			return c
		}
	}

	bs, err := yaml.Marshal(&rootNode)
	if err != nil {
		logrus.Errorf("[autofix] failed to marshal yaml config: %v", err)
		return c
	}

	return string(bs)
}

func setYamlNode(node *yaml.Node, key string, value *yaml.Node) bool {
	keys := strings.SplitN(key, ".", 2)

	switch node.Kind {
	case yaml.DocumentNode:
		return setYamlNode(node.Content[0], key, value)
	case yaml.ScalarNode:
		return setYamlNode(node.Content[1], key, value)
	case yaml.MappingNode:
		for i, n := range node.Content {
			if n.Value == keys[0] {
				if len(keys) == 2 {
					return setYamlNode(node.Content[i+1], keys[1], value)
				} else {
					node.Content[i+1] = value.Content[1]
					return true
				}
			}
		}

		if len(keys) == 2 {
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: keys[0],
			}
			valueNode := &yaml.Node{
				Kind:    yaml.MappingNode,
				Tag:     "!!map",
				Content: []*yaml.Node{},
			}
			node.Content = append(node.Content, keyNode, valueNode)
			return setYamlNode(valueNode, keys[1], value)
		} else {
			node.Content = append(node.Content, value.Content[0], value.Content[1])
			return true
		}
	case yaml.SequenceNode:
		index, err := strconv.Atoi(keys[0])
		if err != nil {
			logrus.Errorf("[yaml] path conversion failed: %v: %v", key, err)
			return false
		}
		if len(node.Content) < index+1 {
			logrus.Errorf("[yaml] path conversion failed: %v: %v", key, err)
			return false
		}

		if len(keys) == 2 {
			return setYamlNode(node.Content[index], keys[1], value)
		} else {
			node.Content[index] = value
			return true
		}
	}

	return false
}

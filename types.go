package main

type TPClashConf struct {
	ProxyMode string

	ClashHome   string
	ClashConfig string
	ClashUI     string
	LocalProxy  bool

	TproxyMark     string
	ClashUser      string
	DirectGroup    string
	HijackIP       string
	HijackDNS      []string
	DisableExtract bool

	Debug bool
}

type ClashConf struct {
	Debug         bool
	EnhancedMode  string
	DNSHost       string
	DNSPort       string
	TProxyPort    string
	FakeIPRange   string
	InterfaceName string
}

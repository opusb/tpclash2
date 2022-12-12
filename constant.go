package main

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/capability.h
const (
	CAP_NET_BIND_SERVICE = 10
	CAP_NET_ADMIN        = 12
	CAP_NET_RAW          = 13
)

const (
	tableFilter = "filter"
	tableMangle = "mangle"
	tableNat    = "nat"

	chainIP4         = "TP_CLASH_V4"
	chainIP4Local    = "TP_CLASH_LOCAL_V4"
	chainIP4DNS      = "TP_CLASH_DNS_V4"
	chainIP4DNSLocal = "TP_CLASH_DNS_LOCAL_V4"
	chainOutput      = "OUTPUT"
	chainPreRouting  = "PREROUTING"
	chainDockerUser  = "DOCKER-USER"

	actionAccept   = "ACCEPT"
	actionReturn   = "RETURN"
	actionTProxy   = "TPROXY"
	actionRedirect = "REDIRECT"
	actionDNat     = "DNAT"
	actionMark     = "MARK"

	systemdResolveGroup = "systemd-resolve"

	defaultTproxyMark  = "666"
	defaultClashUser   = "tpclash"
	defaultDirectGroup = "tpdirect"
)

package main

const (
	tableMangle = "mangle"
	tableNat    = "nat"

	chainIP4         = "TP_CLASH_V4"
	chainIP6         = "TP_CLASH_V6"
	chainIP4Local    = "TP_CLASH_LOCAL_V4"
	chainIP6Local    = "TP_CLASH_LOCAL_V6"
	chainIP4DNS      = "TP_CLASH_DNS_V4"
	chainIP4DNSLocal = "TP_CLASH_DNS_LOCAL_V4"
	chainIP6DNS      = "TP_CLASH_DNS_V6"
	chainIP6DNSLocal = "TP_CLASH_DNS_LOCAL_V6"
	chainOutput      = "OUTPUT"
	chainPreRouting  = "PREROUTING"

	actionReturn   = "RETURN"
	actionTProxy   = "TPROXY"
	actionRedirect = "REDIRECT"
	actionDNat     = "DNAT"
	actionMark     = "MARK"

	tproxyMark = "666"
	clashUser  = "tpclash"
)

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/capability.h
const (
	CAP_NET_BIND_SERVICE = 10
	CAP_NET_ADMIN        = 12
	CAP_NET_RAW          = 13
)

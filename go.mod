module github.com/xflash-panda/server-trojan

go 1.24.4

toolchain go1.24.9

require (
	github.com/sirupsen/logrus v1.9.3
	github.com/urfave/cli/v2 v2.27.7
	github.com/xflash-panda/server-agent-proto v0.0.7
	github.com/xflash-panda/server-client v0.1.4
	github.com/xtls/xray-core v1.250608.0
	google.golang.org/grpc v1.77.0
	google.golang.org/protobuf v1.36.11

)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/apernet/hysteria/core/v2 v2.6.2 // indirect
	github.com/apernet/hysteria/extras/v2 v2.6.2 // indirect
	github.com/apernet/quic-go v0.52.1-0.20250607183305-9320c9d14431 // indirect
	github.com/babolivier/go-doh-client v0.0.0-20201028162107-a76cff4cb8b6 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/database64128/netx-go v0.0.0-20240905055117-62795b8b054a // indirect
	github.com/database64128/tfo-go/v2 v2.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-metro v0.0.0-20211217172704-adc40b04c140 // indirect
	github.com/ghodss/yaml v1.0.1-0.20220118164431-d8423dcdf344 // indirect
	github.com/go-resty/resty/v2 v2.17.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/pprof v0.0.0-20240528025155-186aa0362fba // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.5 // indirect
	github.com/juju/ratelimit v1.0.2 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/miekg/dns v1.1.66 // indirect
	github.com/onsi/ginkgo/v2 v2.19.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pires/go-proxyproto v0.8.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.52.0 // indirect
	github.com/refraction-networking/utls v1.7.3 // indirect
	github.com/riobard/go-bloom v0.0.0-20200614022211-cdc8013cb5b3 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagernet/sing v0.5.1 // indirect
	github.com/sagernet/sing-shadowsocks v0.2.7 // indirect
	github.com/seiflotfy/cuckoofilter v0.0.0-20240715131351-a2f2c23f1771 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/txthinking/runnergroup v0.0.0-20210608031112-152c7c4432bf // indirect
	github.com/txthinking/socks5 v0.0.0-20230325130024-4230056ae301 // indirect
	github.com/v2fly/ss-bloomring v0.0.0-20210312155135-28617310f63e // indirect
	github.com/vishvananda/netlink v1.3.1 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/xtls/reality v0.0.0-20250608132114-50752aec6bfb // indirect
	go.uber.org/goleak v1.3.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	go4.org/netipx v0.0.0-20231129151722-fdeea329fbba // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/net v0.46.1-0.20251013234738-63d1a5100f82 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.37.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251022142026-3a174f9686a8 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gvisor.dev/gvisor v0.0.0-20250428193742-2d800c3129d5 // indirect
	lukechampine.com/blake3 v1.4.1 // indirect
)

replace github.com/xtls/xray-core => github.com/xflash-panda/xray-core v0.0.0-20250711055716-245f06eea47f

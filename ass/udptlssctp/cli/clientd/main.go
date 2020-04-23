package main

import (
	"context"
	"flag"
	"github.com/txthinking/socks5"
	"github.com/xiaokangwang/VLite/ass/licenseroll"
	"github.com/xiaokangwang/VLite/ass/socksinterface"
	"github.com/xiaokangwang/VLite/ass/udpconn2tun"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
	"github.com/xiaokangwang/VLite/interfaces"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var password string
	var address string
	var addressL string
	var LicenseRollOnly bool
	var UseSystemHTTPProxy bool
	var UseSystemSocksProxy bool

	var NetworkBuffering int

	var HTTPDialAddr string

	flag.StringVar(&password, "Password", "", "")
	flag.StringVar(&address, "Address", "", "")
	flag.StringVar(&addressL, "AddressL", "127.0.0.1:1988", "")
	flag.BoolVar(&LicenseRollOnly, "LicenseRollOnly", false, "Show License and Credit")
	flag.BoolVar(&UseSystemHTTPProxy, "UseSystemHTTPProxy", false, "Respect System HTTP Proxy Environment Var HTTP_PROXY HTTPS_PROXY(apply to HTTP transport only)")
	flag.BoolVar(&UseSystemSocksProxy, "UseSystemSocksProxy", false, "Respect System Socks Proxy Environment Var ALL_PROXY(apply to HTTP transport only)")
	flag.IntVar(&NetworkBuffering, "NetworkBuffering", 0, "HTTP Network Buffering Amount(apply to HTTP transport only)")
	flag.StringVar(&HTTPDialAddr, "HTTPDialAddr", "", "If set, HTTP Connections will dial this address instead of requesting DNS(apply to HTTP transport only)")

	flag.Parse()

	if LicenseRollOnly {
		licenseroll.PrintLicense()
		os.Exit(0)
	}
	ctx := context.Background()

	if UseSystemHTTPProxy {
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsHTTPUseSystemHTTPProxy, true)
	}

	if NetworkBuffering != 0 {
		ctxv := &interfaces.ExtraOptionsHTTPNetworkBufferSizeValue{NetworkBufferSize: NetworkBuffering}
		ctx = context.WithValue(ctx, interfaces.ExtraOptionsHTTPNetworkBufferSize, ctxv)
	}

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ctx)
	uc.Up()

	connadp := udpconn2tun.NewUDPConn2Tun(uc.TunnelTxToTun, uc.TunnelRxFromTun)

	socks, err := socks5.NewClassicServer(addressL, "127.0.0.1", "", "", 0, 0, 0, 0)
	if err != nil {
		panic(err)
	}
	socks.Handle = socksinterface.NewSocksHandler(uc, connadp)
	go socks.RunTCPServer()
	go socks.RunUDPServer()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

}

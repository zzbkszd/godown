package shadownet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

var debugMode bool

func dp(a ...interface{}) {
	if debugMode {
		fmt.Println(a...)
	}
}

// 基于shadowsocks的接口实现一个可以直接与ss服务器通信的httpclient

type ShadowConfig struct {
	Ip           string
	Port         int
	Password     []byte
	CryptoMethod string
}

type ShadowTransport struct {
	transport *http.Transport
}

func GetShadowClient(config *ShadowConfig) *http.Client {
	debugMode = true
	st := ShadowTransport{transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
			var zeroDialer net.Dialer
			conn, e = zeroDialer.DialContext(ctx, network, addr)
			return &ShadowWrapConn{conn: conn}, e
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}}
	client := http.Client{Transport: &st}
	return &client
}

func (st *ShadowTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return st.transport.RoundTrip(req)
}

type ShadowWrapConn struct {
	conn     net.Conn // 这个conn是连接到shadowsocks服务器的
	httpAddr net.Addr // 这个是目标网站的地址

	handshake bool
}

func (sc *ShadowWrapConn) doHandshake() error {
	return nil
}

func (sc *ShadowWrapConn) checkHandshake() error {
	if !sc.handshake {
		dp("shadow proxy: should handshake!")
		sc.doHandshake()
	}
	sc.handshake = true
	return nil
}

func (sc *ShadowWrapConn) Read(b []byte) (n int, err error) {
	sc.checkHandshake()
	return sc.conn.Read(b)
}
func (sc *ShadowWrapConn) Write(b []byte) (n int, err error) {
	sc.checkHandshake()
	return sc.conn.Write(b)
}
func (sc *ShadowWrapConn) Close() (err error) {
	return sc.conn.Close()
}
func (sc *ShadowWrapConn) RemoteAddr() net.Addr {
	return sc.conn.RemoteAddr()
}
func (sc *ShadowWrapConn) LocalAddr() net.Addr {
	return sc.conn.LocalAddr()
}
func (sc *ShadowWrapConn) SetDeadline(t time.Time) (err error) {
	return sc.conn.SetDeadline(t)
}
func (sc *ShadowWrapConn) SetWriteDeadline(t time.Time) (err error) {
	return sc.conn.SetWriteDeadline(t)
}
func (sc *ShadowWrapConn) SetReadDeadline(t time.Time) (err error) {
	return sc.conn.SetReadDeadline(t)
}

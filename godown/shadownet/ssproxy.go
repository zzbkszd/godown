package shadownet

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
基于shadowsocks的接口实现一个可以直接与ss服务器通信的httpclient

alpha版本

*/
var debugMode bool = false  // debug输出开关
var directConn bool = false // 直连开关，开启则不使用shadowsocks加密,
var defaultHandshakeMode int = HANDSHAKE_HOST
var DefaultHeader http.Header = map[string][]string{
	"User-Agent": []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"},
}

// short of debug print
func dp(a ...interface{}) {
	if debugMode {
		fmt.Println(a...)
	}
}

type ShadowConfig struct {
	Ip           string
	Port         int
	Password     string
	CryptoMethod string
}

var LocalShadowConfig *ShadowConfig = &ShadowConfig{
	Ip:           "149.28.25.5",
	Port:         6668,
	Password:     "password",
	CryptoMethod: "aes-256-cfb",
}

func GetShadowClient(config *ShadowConfig) *http.Client {
	st := &http.Transport{
		Proxy:                 nil,
		DialContext:           config.shadowDialer,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: st}
	return &client
}

// 不必要的一层封装，直接用shadowDialer替换标准Transport中的DialContext就可以了
// 如果未来要实现连接池的话，还可以用
//type ShadowTransport struct {
//	transport *http.Transport
//}
//
//func (st *ShadowTransport) RoundTrip(req *http.Request) (*http.Response, error) {
//	return st.transport.RoundTrip(req)
//}

/**
shadowsocks加密的拨号器
*/
func (config *ShadowConfig) shadowDialer(ctx context.Context, network, addr string) (conn net.Conn, e error) {
	dp("Dial new conn to ", addr)
	var zeroDialer net.Dialer
	cipher, e := NewSSCipher(config.CryptoMethod, config.Password)
	if directConn {
		conn, e = zeroDialer.DialContext(ctx, network, addr)
	} else {
		ssAddr := fmt.Sprintf("%s:%d", config.Ip, config.Port)
		conn, e = zeroDialer.DialContext(ctx, network, ssAddr)
	}
	if e != nil {
		return nil, e
	}
	return &ShadowWrapConn{conn: conn,
		handshake:     directConn, // 若是开启了直连，就不需要进行握手，否则需要在第一次进行握手
		httpAddr:      addr,
		cryptor:       cipher,
		handshakeMode: defaultHandshakeMode,
		handshakeMu:   &sync.Mutex{}}, e
}

var HANDSHAKE_IP int = 0x01
var HANDSHAKE_HOST int = 0x03

type ShadowWrapConn struct {
	localBuf      *bytes.Buffer
	conn          net.Conn // 这个conn是连接到shadowsocks服务器的
	httpAddr      string   // 这个是目标网站的地址
	cryptor       *SSCipher
	handshakeMode int         // 握手的模式，可选0x01或者0x03
	handshakeMu   *sync.Mutex // mutex for next
	handshake     bool
}

/**
shadowsocks的local端和server端的建立连接握手，而不是socks5的握手。
其实只是socks5协议中的一个步骤，用来指定与远端建立链接
基于ipv4的两种模式:
0x03模式，即传输未经解析的host
+-----+--------+------------+----+
|Proxy|Host Len|Remote Host |Port|
+-----+--------+------------+----+
|0x03 |   len  |  byte[len] |port|
+-----+--------+------------+----+

0x01模式，4个字节的IP地址
+-----+------------+----+
|Proxy|Remote Host |Port|
+-----+------------+----+
|0x01 |  byte[4]   |port|
+-----+------------+----+
*/
func (sc *ShadowWrapConn) doHandshake() error {
	sc.handshake = true
	splited := strings.Split(sc.httpAddr, ":")
	host := splited[0]
	port, _ := strconv.Atoi(splited[1])
	dp("http host:", host, "port:", strconv.Itoa(port))

	header := bytes.NewBuffer(make([]byte, 0))
	// 0x01 mode:
	if sc.handshakeMode == HANDSHAKE_IP {
		header.Write([]byte{0x01})
		remoteAddr := fmt.Sprintf("%s:%d", host, port)
		tcpAddr, e := net.ResolveTCPAddr("tcp", remoteAddr)
		if e != nil {
			panic(e)
		}
		header.Write(tcpAddr.IP.To4()) // feature: ipv4 support only
	} else if sc.handshakeMode == HANDSHAKE_HOST {
		// 0x03 mode:
		header.Write([]byte{0x03, byte(len(host))})
		header.Write([]byte(host))
	}

	// port in the last
	header.Write([]byte{byte(port / 256), byte(port % 256)})

	encrypted := sc.cryptor.encrypt(header.Bytes())
	_, e := sc.conn.Write(encrypted)
	if e != nil {
		panic(e)
	}
	sc.handshakeMu.Unlock()
	return nil
}

func (sc *ShadowWrapConn) checkHandshake() error {
	sc.handshakeMu.Lock()
	if !sc.handshake {
		//dp("shadow proxy: should handshake!")
		return sc.doHandshake()
	}
	sc.handshakeMu.Unlock()
	return nil
}

func (sc *ShadowWrapConn) initLocalBuf() {
	sc.localBuf = bytes.NewBuffer(make([]byte, 0))
}

/**
要先读到内存缓冲区，再从缓冲区读出解密的数据
不然读取指定长度的数据会导致解密失败
*/
func (sc *ShadowWrapConn) readToBuf() error {
	if sc.localBuf == nil {
		sc.initLocalBuf()
	}
	// 只在缓存区没有数据的时候去读取，否则当服务器写完之后会阻塞到超时
	if sc.localBuf.Len() != 0 {
		return nil
	}
	buf := make([]byte, 8192)
	n, err := sc.conn.Read(buf)
	if err != nil {
		return err
	}
	decrypt, _ := sc.cryptor.decrypt(buf[:n])
	n, err = sc.localBuf.Write(decrypt)
	if err != nil {
		return err
	}
	return nil
}
func (sc *ShadowWrapConn) Read(b []byte) (n int, err error) {
	if directConn {
		return sc.conn.Read(b)
	}
	sc.checkHandshake()
	err = sc.readToBuf()
	if err != nil {
		return
	}
	n, err = sc.localBuf.Read(b)
	//dp("read content\n", fmt.Sprintf("%s", b))
	return
}
func (sc *ShadowWrapConn) Write(b []byte) (n int, err error) {
	if directConn {
		return sc.conn.Write((b))
	}
	sc.checkHandshake()
	//dp("request content:", fmt.Sprintf("\n%s", b))
	buf := sc.cryptor.encrypt(b)
	//dp("request content encrypted:", fmt.Sprintf("%x", buf))
	n, err = sc.conn.Write(buf)
	return
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

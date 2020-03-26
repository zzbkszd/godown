package shadownet

import (
	"context"
	"crypto/aes"
	"encoding/base64"
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
)

/**
自动抓取shadowsocks免费代理
*/

type ShadowProxyInfo struct {
	config   *ShadowConfig
	enable   bool      // 是否可用
	lastTest time.Time // 最后测试时间
	delay    int       // 延迟毫秒数
}

func (spi *ShadowProxyInfo) test() bool {
	client := GetShadowClient(spi.config)
	if _, e := client.Get("https://www.google.com"); e == nil {
		dp("Test success")
		return true
	} else {
		dp("Test fail")
		panic(e)
		return false
	}

}

func GetShadowPool() {
	fmt.Println("[FreeSS] start to get free shadowsocks server")
	lncnPool := fetchLncn()
	if len(lncnPool) == 0 {
		panic(fmt.Errorf("Network error!"))
	}

	fmt.Println("[FreeSS] fetch lncn config done, got", len(lncnPool), "servers")
	fmt.Println("[FreeSS] start fetching sstool configs")
	shadowClient := GetShadowClient(lncnPool[0].config)
	toolsPool := fetchSStool(shadowClient)

	fmt.Println("[FreeSS] fetch sstool config done, got", len(toolsPool), "servers")
	file, _ := os.OpenFile("configs.json", os.O_CREATE, 0777)
	defer file.Close()
	file.Write([]byte("["))
	for _, info := range lncnPool {
		printConfig(info.config, file)
	}
	for _, info := range toolsPool {
		printConfig(info.config, file)
	}
	file.Write([]byte("]"))
	fmt.Println("[FreeSS] config save to configs.json!")
}

func printConfig(config *ShadowConfig, writer io.Writer) {
	str := fmt.Sprintf(`{ "server": "%s", "server_port": %d, "password": "%s", "method": "%s" }`,
		config.Ip, config.Port, config.Password, config.CryptoMethod)
	writer.Write([]byte(str))
	writer.Write([]byte(",\n"))
}

/**
https://lncn.org/
大陆直连！
*/
func fetchLncn() []ShadowProxyInfo {
	api := "https://lncn.org/api/SSR"
	req, err := http.NewRequest(http.MethodPost, api, nil)
	if err != nil {
		panic(err)
	}
	req.Header = DefaultHeader
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
				var zeroDialer net.Dialer
				conn, e = zeroDialer.DialContext(ctx, network, "162.159.208.87:443")
				return
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	context, _ := ioutil.ReadAll(resp.Body)
	ssrs64 := gjson.Get(string(context), "ssrs")
	key64 := gjson.Get(string(context), "ip")
	key, _ := base64.RawStdEncoding.DecodeString(key64.String())
	ssrs, _ := base64.StdEncoding.DecodeString(ssrs64.String())
	decode, err := aesEcbDecode(ssrs, key)
	res := gjson.Get(decode, "..0")
	result := make([]ShadowProxyInfo, 0)
	for _, ssr := range res.Array() {
		ssr := ssr.Map()["ssr"].Map()
		spi := ShadowProxyInfo{
			config: &ShadowConfig{
				Ip:           ssr["ip"].String(),
				Port:         int(ssr["port"].Int()),
				Password:     ssr["password"].String(),
				CryptoMethod: ssr["method"].String(),
			},
			enable:   false,
			lastTest: time.Time{},
			delay:    0,
		}
		//fmt.Println(spi.config)
		//if spi.test() {
		result = append(result, spi)
		//}
	}
	if err != nil {
		panic(err)
	}
	return result
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
func aesEcbDecode(encrypted []byte, key []byte) (string, error) {
	//key只能是 16 24 32长度
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	//返回加密结果\
	decryptData := make([]byte, len(encrypted))
	blockSize := aes.BlockSize
	//分组分块加密
	for s, e := 0, blockSize; s < len(encrypted); s, e = s+blockSize, e+blockSize {
		block.Decrypt(decryptData[s:e], encrypted[s:e])
	}
	return string(PKCS7UnPadding(decryptData)), nil
}

/**
https://ssrtool.us/tool/free_ssr
*/
func fetchSStool(client *http.Client) []ShadowProxyInfo {
	client.Jar, _ = cookiejar.New(nil)
	initReq, _ := http.NewRequest(http.MethodGet, "https://ssrtool.us/tool/free_ssr", nil)
	initReq.Header.Add("Host", "ssrtool.us")
	initReq.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	initReq.Header.Add("Connection", "keep-alive")
	initReq.Header.Add("Accept", "*/*")
	initResp, _ := client.Do(initReq)
	fmt.Println(initResp.Cookies())
	fmt.Println(initResp.Header)
	c, _ := ioutil.ReadAll(initResp.Body)
	fmt.Println(string(c))

	//jsessionReq, _ := http.NewRequest(http.MethodGet, "")

	api := "https://ssrtool.us/tool/api/free_ssr?page=1&limit=50"
	req, _ := http.NewRequest(http.MethodGet, api, nil)
	fmt.Println(req.Cookies())
	//req.Header.Set("cookie", "__cfduid=ddcea3275e167702323836bd6249ed3e81585019072; JSESSIONID=3E0B93BF7615851A8EE6BAE1F4B9DA1F; _ga=GA1.2.1122442270.1585019283; _gid=GA1.2.906306432.1585019283; _gat_gtag_UA_114706424_1=1")
	resp, _ := client.Do(req)
	context, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(context))
	data := gjson.Get(string(context), `data.#(protocol=="origin")#`)
	dp("fetch proxys:", len(data.Array()))
	result := make([]ShadowProxyInfo, 0)
	for _, res := range data.Array() {
		vm := res.Map()
		conf := ShadowConfig{
			Ip:           vm[`server`].String(),
			Port:         int(vm[`server_port`].Int()),
			Password:     vm[`password`].String(),
			CryptoMethod: vm[`method`].String(),
		}
		spi := ShadowProxyInfo{
			config:   &conf,
			enable:   false,
			lastTest: time.Time{},
		}
		if spi.test() {
			result = append(result, spi)
		}
	}
	return result
}

/**
这玩意的代码混淆太尼玛麻烦了，暂时放弃
返回数据是AES-ECB模式的
*/
func fetchFreess() {
	api := "https://free-ss.site/data3.php"
	req, _ := http.NewRequest(http.MethodGet, api, nil)
	client := GetShadowClient(LocalShadowConfig)
	resp, _ := client.Do(req)
	fmt.Println(resp)
}

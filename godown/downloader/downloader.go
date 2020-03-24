package downloader

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/zzbkszd/godown/godown/shadownet"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

type Downloader interface {
	Download(url string, dist string) error
}

// 下载器的抽象接口
type AbstractDownloader struct {
	name   string
	Client *http.Client
}

// Implement for interface Downloader
func (d *AbstractDownloader) Download(url string, dist string) error {
	return fmt.Errorf("Not Implement Function")
}

// 初始化网络等信息
func (d *AbstractDownloader) Init() {
	if d.Client == nil {
		//d.Client = &http.Client{}
		d.Client = shadownet.GetShadowClient(shadownet.LocalShadowConfig)
		//proxyUrl, _ := url.Parse("socks5://127.0.0.1:1080")
		//d.Client = &http.Client{
		//	Transport: &http.Transport{
		//		Proxy: http.ProxyURL(proxyUrl),
		//	},
		//}
	}
}

// 预先创建目录
func (d *AbstractDownloader) PrepareDist(dist string) {
	dir := path.Dir(dist)
	os.MkdirAll(dir, 0777)
}

// 拉取Content-Length
func (d *AbstractDownloader) FetchSize(req *http.Request) int {
	resp, err := d.Client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	cl, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	return cl
}

// 拉取文本内容
func (d *AbstractDownloader) FetchText(req *http.Request) string {
	resp, err := d.Client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return string(buf)
}

// 基准的http下载方法
func (d *AbstractDownloader) HttpDown(req *http.Request, dist string) {
	resp, err := d.Client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	d.PrepareDist(dist)
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	if e != nil {
		panic(e)
	}
	_, e = io.Copy(distFile, resp.Body)
	if e != nil {
		panic(e)
	}
}

/**
这个的实现就是为了能够调用HttpDown，避免抽象类的Download方法没有实现的问题
*/
type HttpDownloader struct {
	AbstractDownloader
}

func (d *HttpDownloader) Download(urlstr string, dist string) error {
	d.Init()
	resp, err := d.Client.Do(quickRequest(http.MethodGet, urlstr, nil))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	d.PrepareDist(dist)
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	cl, e := strconv.Atoi(resp.Header.Get("Content-Length"))
	bar := pb.Full.Start64(int64(cl))
	if e != nil {
		panic(e)
	}
	pr := bar.NewProxyReader(resp.Body)
	_, e = io.Copy(distFile, pr)
	bar.Finish()
	if e != nil {
		panic(e)
	}
	return nil
}

/** **************************
Some useful utils
************************** **/
// simple and typical http request
func quickRequest(method string, urlStr string, headers http.Header) (req *http.Request) {
	reqUrl, e := url.Parse(urlStr)
	if e != nil {
		panic(e)
	}
	if headers == nil {
		headers = shadownet.DefaultHeader
	} else {
		headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	}
	req = &http.Request{Method: http.MethodGet, URL: reqUrl, Header: headers}

	return req
}

func getParentUrl(base string) string {
	parent := strings.Split(base, "/")
	return strings.Join(parent[:len(parent)-1], "/")
}

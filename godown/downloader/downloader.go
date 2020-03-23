package downloader

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
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
		d.Client = &http.Client{}
		//d.Client = shadownet.GetShadowClient(shadownet.LocalShadowConfig)
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

// 拉取html页面
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
	fmt.Println("response code:", resp.Status)
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
}

type HttpDownloader struct {
	AbstractDownloader
}

func (d *HttpDownloader) Download(urlstr string, dist string) error {
	d.Init()
	url, e := url.Parse(urlstr)
	if e != nil {
		panic(e)
	}
	request := http.Request{Method: "Get", URL: url}
	d.HttpDown(&request, dist)
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
		headers = http.Header{
			"User-Agent": []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36"},
		}
	} else {
		headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	}
	req = &http.Request{Method: http.MethodGet, URL: reqUrl, Header: headers}

	return req
}

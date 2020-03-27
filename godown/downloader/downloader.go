package downloader

import (
	"fmt"
	"github.com/zzbkszd/godown/godown"
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

/**
下载器
下载器只用作下载单个数据
数据列表的爬取工作是构造collect的工作。
*/
type Downloader interface {
	Download(url string, dist string) error
	godown.ProgressAble
}

/**
下载器的抽象接口， 实现了Downloader接口，但是没有实现Download方法
该接口主要是实现了统一的进度管理功能，避免了进度条的显示混乱
*/
type AbstractDownloader struct {
	name   string
	Client *http.Client
	// 关于进度的成员变量：
	godown.CommonProgress
}

// Implement for interface Downloader
func (d *AbstractDownloader) Download(url string, dist string) error {
	return fmt.Errorf("Not Implement Function")
}

// 初始化网络等信息
// 默认使用http.DefaultClient，如需代理在外层指定，可以直接赋值Client
func (d *AbstractDownloader) Init() {
	if d.Client == nil {
		d.Client = http.DefaultClient
	}
}

// 预先创建目录
func (d *AbstractDownloader) PrepareDist(dist string) {
	dir := path.Dir(dist)
	os.MkdirAll(dir, 0777)
}

// 拉取Content-Length
func (d *AbstractDownloader) FetchSize(req *http.Request) (int, error) {
	resp, err := d.Client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return 0, err
	}
	cl, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	return cl, nil
}

// 拉取文本内容
func (d *AbstractDownloader) FetchText(req *http.Request) (string, error) {
	resp, err := d.Client.Do(req)
	if resp.Body == nil {
		return "", fmt.Errorf("Abstract Downloader: fetch text fail: no response body")
	}
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// 基准的http下载方法
func (d *AbstractDownloader) HttpDown(req *http.Request, dist string) error {
	resp, err := d.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d.PrepareDist(dist)
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	if e != nil {
		return e
	}
	_, e = io.Copy(distFile, resp.Body)
	if e != nil {
		return e
	}
	return nil
}

/**
这个的实现就是为了能够调用HttpDown，避免抽象类的Download方法没有实现的问题
*/
type HttpDownloader struct {
	AbstractDownloader
	Header http.Header
}

type ProgressReader struct {
	io.Reader
	downloader *AbstractDownloader
}

// Read reads bytes from wrapped reader and add amount of bytes to progress bar
func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.downloader.UpdateProgress(int64(n))
	return
}

// Close the wrapped reader when it implements io.Closer
func (r *ProgressReader) Close() (err error) {
	if closer, ok := r.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return
}

func (d *HttpDownloader) Download(urlstr string, dist string) error {
	d.Init()
	resp, err := d.Client.Do(quickRequest(http.MethodGet, urlstr, d.Header))
	if err != nil {
		return (err)
	}
	defer resp.Body.Close()
	d.PrepareDist(dist)
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	cl, e := strconv.Atoi(resp.Header.Get("Content-Length"))
	d.InitProgress(int64(cl), true)
	defer d.CloseProgress()
	if e != nil {
		return (e)
	}
	pr := &ProgressReader{resp.Body, &d.AbstractDownloader}
	_, e = io.Copy(distFile, pr)
	if e != nil {
		return (e)
	}
	return nil
}

/** **************************
Some useful utils
************************** **/
// simple and typical http request
func quickRequest(method string, urlStr string, headers http.Header) (req *http.Request) {
	if headers == nil {
		headers = shadownet.DefaultHeader
	} else {
		headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	}
	req, _ = http.NewRequest(http.MethodGet, urlStr, nil)
	req.Header = headers
	return req
}

func getParentUrl(base string) string {
	parent := strings.Split(base, "/")
	return strings.Join(parent[:len(parent)-1], "/")
}

func GetUrlFileName(base string) string {
	if strings.HasPrefix(base, "http") {
		if u, e := url.Parse(base); e == nil {
			path := strings.Split(u.Path, "/")
			return path[len(path)-1]
		}
	}
	s, e := strings.LastIndex(base, "/"), strings.Index(base, "?")
	if s == -1 {
		s = 0
	}
	if e > 0 {
		return base[s:e]
	} else {
		return base[s:]
	}

}

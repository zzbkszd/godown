package godown

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
)

type Downloader interface {
	Download(url string, dist string)
}

// 下载器的抽象接口
type AbstractDownloader struct {
	name   string
	Client *http.Client
}

func (d *AbstractDownloader) Download(url string, dist string) {
	panic("Not Implemented Function")
}

func (d *AbstractDownloader) Init() {
	if d.Client == nil {
		d.Client = &http.Client{}
	}
}

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

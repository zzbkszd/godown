package downloader

import (
	"fmt"
	"github.com/zzbkszd/godown/common"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

/**
下载器
下载器只用作下载单个数据
数据列表的爬取工作是构造collect的工作。
以下实现了几个基本的下载器，以供更多的下载器来调用
*/
type Downloader interface {
	SetClient(client *http.Client)
	Download(url string, dist string) (string, error) // 返回最终下载的文件的路径
	common.ProgressAble
}

/**
下载器的抽象接口， 实现了Downloader接口，但是没有实现Download方法
该接口主要是实现了统一的进度管理功能，避免了进度条的显示混乱
*/
type AbstractDownloader struct {
	name   string
	Client *http.Client
	// 关于进度的成员变量：
	common.CommonProgress
}

// Implement for interface Downloader
func (d *AbstractDownloader) Download(url string, dist string) (string, error) {
	return "", fmt.Errorf("Not Implement Function")
}

// 初始化网络等信息
// 默认使用http.DefaultClient，如需代理在外层指定，可以直接赋值Client
func (d *AbstractDownloader) Init() {
	if d.Client == nil {
		d.Client = http.DefaultClient
	}
}

func (d *AbstractDownloader) SetClient(client *http.Client) {
	d.Client = client
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
	if err != nil {
		return "", err
	}
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
	if resp.StatusCode > 399 {
		fmt.Println("[http text]" + string(buf))
		return "", fmt.Errorf("Error http response code: %d", resp.StatusCode)
	}
	return string(buf), nil
}

// 基准的http下载方法
func (d *AbstractDownloader) HttpDown(req *http.Request, dist string) error {
	resp, err := d.Client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode > 399 {
		//buf, _ := ioutil.ReadAll(resp.Body)
		//fmt.Println("[http down]" + string(buf))
		return fmt.Errorf("Error http response code: %d", resp.StatusCode)
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

func (d *AbstractDownloader) HttpDownWithProgress(req *http.Request, dist string) error {
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
	pr := &ProgressReader{resp.Body, &d.CommonProgress}
	_, e = io.Copy(distFile, pr)
	if e != nil {
		return e
	}
	return nil
}

/**
可以更新进度条的reader
*/
type ProgressReader struct {
	io.Reader
	progress *common.CommonProgress
}

// Read reads bytes from wrapped reader and add amount of bytes to progress bar
func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.progress.UpdateProgress(int64(n))
	return
}

/**
这个的实现就是为了能够调用HttpDown，避免抽象类的Download方法没有实现的问题
*/
type HttpDownloader struct {
	AbstractDownloader
	Header http.Header
}

var DefaultHttpDownloader = HttpDownloader{
	AbstractDownloader: AbstractDownloader{
		Client: http.DefaultClient,
		CommonProgress: common.CommonProgress{
			DisplayProgress: false,
			DisplayOnUpdate: true,
		},
	},
}

func (d *HttpDownloader) Download(urlstr string, dist string) (string, error) {
	d.Init()
	resp, err := d.Client.Do(QuickRequest(http.MethodGet, urlstr, d.Header))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	d.PrepareDist(dist)
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	cl, e := strconv.Atoi(resp.Header.Get("Content-Length"))
	d.InitProgress(int64(cl), true)
	defer d.CloseProgress()
	if e != nil {
		return "", e
	}
	pr := &ProgressReader{resp.Body, &d.CommonProgress}
	_, e = io.Copy(distFile, pr)
	if e != nil {
		return "", e
	}
	return dist, nil
}

/**
一个更好的Http下载器，适用于下载大文件
支持分块下载
*/
type MultipartHttpDownloader struct {
	AbstractDownloader
	Headers http.Header
	Threads int
}

var MB = 1024 * 1024

func (d *MultipartHttpDownloader) Download(urlstr string, dist string) (string, error) {
	d.PrepareDist(dist)
	// 先检测是否可以使用分块下载
	h1 := http.Header{}
	h1.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.162 Safari/537.36")
	h1.Add("Range", "bytes=0-10")
	resp, err := d.Client.Do(QuickRequest(http.MethodGet, urlstr, h1))
	if err != nil {
		return "", err
	}
	code := resp.StatusCode
	contentLength := 0
	parts := 1
	if code == 206 { // support multipart transport
		cr := resp.Header.Get("Content-Range")
		contentLength, err := strconv.Atoi(cr[strings.LastIndex(cr, "/")+1:])
		if err != nil {
			return "", err
		}
		if contentLength < 5*MB {
			parts = 1
		} else {
			parts = contentLength/MB/5 + 1
		}
	} else {
		fmt.Println("[Multipart Downloader] server not support, use httpdown method")
	}
	if parts == 1 {
		fmt.Println("[Multipart Downloader] quick down!")
		err := d.HttpDownWithProgress(QuickRequest(http.MethodGet, urlstr, nil), dist)
		return dist, err
	}
	dir := path.Dir(dist)
	cache, err := ioutil.TempDir(dir, "multipart*")
	tasks := []func() error{}
	fmt.Println("[MultipartDownloader]content length is:", contentLength)
	d.InitProgress(int64(contentLength), true)
	for i := 0; i < parts; i++ {
		s, e := i*5*MB, (i+1)*5*MB-1
		partHeader := http.Header{}
		partHeader.Add("Range", fmt.Sprintf("bytes=%d-%d", s, e))
		partDist := path.Join(cache, fmt.Sprintf("%d.part", i))
		tasks = append(tasks, func() error {
			err = d.HttpDownWithProgress(QuickRequest(http.MethodGet, urlstr, partHeader), partDist)
			if err != nil {
				return err
			} else {
				return nil
			}
		})
	}
	if d.Threads == 0 {
		d.Threads = 5
	}
	cycle := common.MultiTaskCycle{
		Threads:   d.Threads,
		TryOnFail: true,
	}
	cycle.CostTasks(tasks)
	d.CloseProgress()
	d.combineParts(cache, dist, parts)
	return dist, nil
}

func (d *MultipartHttpDownloader) combineParts(partDir, dist string, parts int) {
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	if e != nil {
		panic(e)
	}
	for i := 0; i < parts; i++ {
		partDist := path.Join(partDir, fmt.Sprintf("%d.part", i))
		partFile, e := os.OpenFile(partDist, os.O_RDONLY, 0777)
		if e != nil {
			panic(e)
		}
		io.Copy(distFile, partFile)
		partFile.Close()
		os.Remove(partDist)
	}
	distFile.Close()
	os.Remove(partDir)
}

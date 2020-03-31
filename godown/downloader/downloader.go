package downloader

import (
	"fmt"
	"github.com/zzbkszd/godown/godown/common"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
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

func (d *HttpDownloader) Download(urlstr string, dist string) (string, error) {
	d.Init()
	resp, err := d.Client.Do(quickRequest(http.MethodGet, urlstr, d.Header))
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
m3u8 下载器
暂不支持加密格式，未进行格式转换
支持多线程并发下载，默认线程数为5
todo 已知设计BUG： 当因网络链接之类的问题导致下载确实无法进行时会无限次数重试。
todo 使用common中的线程循环来简化代码 - 未测试
*/
type M3u8Downloader struct {
	AbstractDownloader
	Threads int
	//tsLock  *sync.Mutex
}

//type tsTask struct {
//	baseUrl, distDir, tsUrl string
//}

func (d *M3u8Downloader) Download(urlstr, dist string) (string, error) {
	d.Init()
	if d.Threads == 0 {
		d.Threads = 5
	}
	tsdir := path.Join(path.Dir(dist), "ts"+strconv.Itoa(int(time.Now().Unix())))
	d.PrepareDist(tsdir)
	m3u8File, err := d.FetchText(quickRequest(http.MethodGet, urlstr, nil))
	if err != nil {
		return "", err
	}
	tsList := d.parseTsList(m3u8File)
	d.doDownload(tsList, urlstr, tsdir)
	d.combinTs(tsList, dist, tsdir)
	return dist, nil
}

func (d *M3u8Downloader) combinTs(tsList []string, dist, tsdir string) {
	fmt.Printf("[M3U8 Downloader] start combin ts data \n")
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	defer distFile.Close()
	if e != nil {
		panic(e)
	}
	for _, name := range tsList {
		tsPath := path.Join(tsdir, GetUrlFileName(name))
		tsFile, e := os.OpenFile(tsPath, os.O_RDONLY, 0777)
		if e != nil {
			panic(e)
		}
		io.Copy(distFile, tsFile)
		tsFile.Close()
		os.Remove(tsPath)
	}

}

func (d *M3u8Downloader) doDownload(tsList []string, baseUrl, tsdir string) {
	parent := strings.Split(baseUrl, "/")
	base := strings.Join(parent[:len(parent)-1], "/")
	d.InitProgress(int64(len(tsList)), false)
	defer d.CloseProgress()
	taskSet := common.TaskSet{}
	for _, ts := range tsList {
		keyUrl := ts
		taskSet = taskSet.Add(func() error {
			tsUrl := strings.Join([]string{base, keyUrl}, "/")
			tsDist := path.Join(tsdir, GetUrlFileName(keyUrl))
			err := d.HttpDown(quickRequest(http.MethodGet, tsUrl, nil), tsDist)
			if err != nil {
				return err
			}
			d.UpdateProgress(1)
			return nil
		})
	}
	cycle := common.MultiTaskCycle{
		Threads:   d.Threads,
		TryOnFail: true,
	}
	cycle.CostTasks(taskSet)
}

//func (d *M3u8Downloader) downloadGoChan(group *sync.WaitGroup, tsCh chan *tsTask, doneCh chan int) {
//	for {
//		select {
//		case ts := <-tsCh:
//			tsUrl := strings.Join([]string{ts.baseUrl, ts.tsUrl}, "/")
//			tsDist := path.Join(ts.distDir, GetUrlFileName(ts.tsUrl))
//			if err := d.HttpDown(quickRequest(http.MethodGet, tsUrl, nil), tsDist); err != nil {
//				tsCh <- ts
//			} else {
//				d.UpdateProgress(1)
//				group.Done()
//			}
//		case <-doneCh:
//			break
//		}
//	}
//}

func (d *M3u8Downloader) parseTsList(m3u8 string) []string {
	baseList := strings.Split(m3u8, "\n")
	distList := []string{}
	for _, line := range baseList {
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		} else {
			distList = append(distList, line)
		}
	}
	return distList
}

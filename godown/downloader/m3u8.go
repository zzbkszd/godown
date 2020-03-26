package downloader

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

/**
m3u8 下载器
暂不支持加密格式，未进行格式转换
*/
/**
todo： 失败任务的重试
*/
type M3u8Downloader struct {
	AbstractDownloader
	tsList  []string // todo 用切片模拟队列是一件性能很低下的操作，考虑使用缓冲管道来处理
	tsLock  *sync.Mutex
	Threads int
}
type tsTask struct {
	baseUrl, distDir, tsUrl string
}

func (d *M3u8Downloader) Download(urlstr, dist string) error {
	d.Init()
	if d.Threads == 0 {
		d.Threads = 5
	}
	tsdir := path.Join(path.Dir(dist), "ts")
	d.PrepareDist(tsdir)
	m3u8File, err := d.FetchText(quickRequest(http.MethodGet, urlstr, nil))
	if err != nil {
		return err
	}
	tsList := d.parseTsList(m3u8File)
	d.doDownload(tsList, urlstr, tsdir)
	d.combinTs(tsList, dist, tsdir)
	return nil
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
	bar := pb.StartNew(len(tsList))
	defer bar.Finish()
	tsCh := make(chan *tsTask, d.Threads*2)
	doneCh := make(chan int, d.Threads)
	waitGroup := &sync.WaitGroup{}
	for i := 0; i < d.Threads; i++ {
		go d.downloadGoChan(bar, waitGroup, tsCh, doneCh)
	}
	waitGroup.Add(len(tsList))
	for _, ts := range tsList {
		tsCh <- &tsTask{
			baseUrl: base,
			distDir: tsdir,
			tsUrl:   ts,
		}
	}
	waitGroup.Wait()
	for i := 0; i < d.Threads; i++ {
		doneCh <- 1
	}
}
func (d *M3u8Downloader) popTs() string {
	d.tsLock.Lock()
	tsName := d.tsList[0]
	d.tsList = d.tsList[1:]
	d.tsLock.Unlock()
	return tsName
}
func (d *M3u8Downloader) pushTs(ts string) {
	d.tsLock.Lock()
	d.tsList = append(d.tsList, ts)
	d.tsLock.Unlock()
}

func (d *M3u8Downloader) downloadGoChan(bar *pb.ProgressBar, group *sync.WaitGroup,
	tsCh chan *tsTask, doneCh chan int) {
	for {
		select {
		case ts := <-tsCh:
			tsUrl := strings.Join([]string{ts.baseUrl, ts.tsUrl}, "/")
			tsDist := path.Join(ts.distDir, GetUrlFileName(ts.tsUrl))
			if err := d.HttpDown(quickRequest(http.MethodGet, tsUrl, nil), tsDist); err != nil {
				tsCh <- ts
				fmt.Println("DEBUG: download fail! add to chan:", ts.tsUrl)
			} else {
				bar.Increment()
				group.Done()
			}

		case <-doneCh:
			fmt.Println("DEBUG: download go chan closed!")
			break
		}
	}
}

/**
已废弃： 使用chan来代替低性能的队列
*/
func (d *M3u8Downloader) downloadGo(baseUrl string, tsdir string,
	bar *pb.ProgressBar, group *sync.WaitGroup) {
	for len(d.tsList) > 0 {
		tsName := d.popTs()
		tsUrl := strings.Join([]string{baseUrl, tsName}, "/")
		tsDist := path.Join(tsdir, GetUrlFileName(tsName))
		err := d.HttpDown(quickRequest(http.MethodGet, tsUrl, nil), tsDist)
		if err != nil {
			d.pushTs(tsName)
		}
		bar.Increment()
	}
	group.Done()
}

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

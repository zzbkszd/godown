package downloader

import (
	"fmt"
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
支持多线程并发下载，默认线程数为5
已知设计BUG： 当因网络链接之类的问题导致下载确实无法进行时会无限次数重试。
*/
type M3u8Downloader struct {
	AbstractDownloader
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
	d.InitProgress(int64(len(tsList)), false)
	defer d.CloseProgress()
	tsCh := make(chan *tsTask, d.Threads*2)
	doneCh := make(chan int, d.Threads)
	waitGroup := &sync.WaitGroup{}
	for i := 0; i < d.Threads; i++ {
		go d.downloadGoChan(waitGroup, tsCh, doneCh)
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

func (d *M3u8Downloader) downloadGoChan(group *sync.WaitGroup, tsCh chan *tsTask, doneCh chan int) {
	for {
		select {
		case ts := <-tsCh:
			tsUrl := strings.Join([]string{ts.baseUrl, ts.tsUrl}, "/")
			tsDist := path.Join(ts.distDir, GetUrlFileName(ts.tsUrl))
			if err := d.HttpDown(quickRequest(http.MethodGet, tsUrl, nil), tsDist); err != nil {
				tsCh <- ts
				fmt.Println("DEBUG: download fail! add to chan:", ts.tsUrl)
			} else {
				d.UpdateProgress(1)
				group.Done()
			}
		case <-doneCh:
			fmt.Println("DEBUG: download go chan closed!")
			break
		}
	}
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

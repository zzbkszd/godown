package downloader

import (
	"github.com/cheggaaa/pb/v3"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
)

type M3u8Downloader struct {
	AbstractDownloader
	tsList  []string // todo 用切片模拟队列是一件性能很低下的操作
	tsLock  *sync.Mutex
	Threads int
}

func (d *M3u8Downloader) Download(urlstr, dist string) error {
	d.Init()
	if d.Threads == 0 {
		d.Threads = 5
	}
	tsdir := path.Join(path.Dir(dist), "ts")
	d.PrepareDist(tsdir)
	m3u8File := d.FetchText(quickRequest(http.MethodGet, urlstr, nil))
	tsList := d.parseTsList(m3u8File)
	d.doDownload(tsList, urlstr, tsdir)
	d.combinTs(tsList, dist, tsdir)
	return nil

}

func (d *M3u8Downloader) combinTs(tsList []string, dist, tsdir string) {
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	defer distFile.Close()
	if e != nil {
		panic(e)
	}
	for _, name := range tsList {
		tssrc := path.Join(tsdir, name)
		tsFile, e := os.OpenFile(tssrc, os.O_RDONLY, 0777)
		defer tsFile.Close()
		if e != nil {
			panic(e)
		}
		io.Copy(distFile, tsFile)
		defer os.Remove(tssrc)
	}

}

func (d *M3u8Downloader) doDownload(tsList []string, baseUrl, tsdir string) {
	parent := strings.Split(baseUrl, "/")
	base := strings.Join(parent[:len(parent)-1], "/")
	bar := pb.StartNew(len(tsList))
	defer bar.Finish()
	d.tsLock = &sync.Mutex{}
	d.tsList = tsList
	waitGroup := &sync.WaitGroup{}
	for i := 0; i < d.Threads; i++ {
		go d.downloadGo(base, tsdir, bar, waitGroup)
		waitGroup.Add(1)
	}
	waitGroup.Wait()
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

func (d *M3u8Downloader) downloadGo(baseUrl string, tsdir string,
	bar *pb.ProgressBar, group *sync.WaitGroup) {
	for len(d.tsList) > 0 {
		tsName := d.popTs()
		tsUrl := strings.Join([]string{baseUrl, tsName}, "/")
		tsDist := path.Join(tsdir, tsName)
		d.HttpDown(quickRequest(http.MethodGet, tsUrl, nil), tsDist)
		bar.Increment()
	}
	group.Done()
}

func (d *M3u8Downloader) parseTsList(m3u8 string) []string {
	baseList := strings.Split(m3u8, "\n")
	distList := []string{}
	for _, line := range baseList {
		if len(line) == 0 || strings.Contains(line, "#") {
			continue
		} else {
			distList = append(distList, line)
		}
	}
	return distList
}

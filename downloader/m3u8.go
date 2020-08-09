package downloader

import (
	"fmt"
	"github.com/zzbkszd/godown/common"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

/**
m3u8 下载器
暂不支持加密格式，未进行格式转换
支持多线程并发下载，默认线程数为5
todo 已知设计BUG： 当因网络链接之类的问题导致下载确实无法进行时会无限次数重试。
*/
type M3u8Downloader struct {
	AbstractDownloader
	Threads int
}

func (d *M3u8Downloader) Download(urlstr, dist string) (string, error) {
	d.Init()
	if d.Threads == 0 {
		d.Threads = 5
	}
	d.PrepareDist(dist)
	tsdir, err := ioutil.TempDir(path.Dir(dist), "ts*")
	if err != nil {
		return "", err
	}
	m3u8File, err := d.FetchText(QuickRequest(http.MethodGet, urlstr, nil))
	if err != nil {
		return "", err
	}
	tsList := d.parseTsList(m3u8File)
	d.doDownload(tsList, urlstr, tsdir)
	err = d.combinTs(tsList, dist, tsdir)
	if err != nil {
		return "", err
	}
	return dist, nil
}

func (d *M3u8Downloader) combinTs(tsList []string, dist, tsdir string) error {
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
		_, err := io.Copy(distFile, tsFile)
		if err != nil {
			panic(err)
		}
		tsFile.Close()
		os.Remove(tsPath)
	}
	finfo, err := distFile.Stat()
	if err != nil {
		return err
	}
	fileLength := finfo.Size()
	if fileLength < 1024*1024 {
		return fmt.Errorf("file size too small: %s", dist)
	}
	os.Remove(tsdir)
	return nil
}

func (d *M3u8Downloader) doDownload(tsList []string, baseUrl, tsdir string) {
	parent := strings.Split(baseUrl, "/")
	base := strings.Join(parent[:len(parent)-1], "/")
	d.InitProgress(int64(len(tsList)), false)
	defer d.CloseProgress()
	taskSet := make([]func() error, 0)
	for _, ts := range tsList {
		keyUrl := ts
		taskSet = append(taskSet, func() error {
			tsUrl := strings.Join([]string{base, keyUrl}, "/")
			tsDist := path.Join(tsdir, GetUrlFileName(keyUrl))
			err := d.HttpDown(QuickRequest(http.MethodGet, tsUrl, nil), tsDist)
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

func (d *M3u8Downloader) parseTsList(m3u8 string) []string {
	baseList := strings.Split(m3u8, "\n")
	distList := []string{}
	for _, line := range baseList {
		if strings.Contains(line, "head") {
			fmt.Println("[DEBUG] guess this is a page:")
			fmt.Println(m3u8)
		}
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		} else {
			distList = append(distList, line)
		}
	}
	return distList
}

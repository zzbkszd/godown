package downloader

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

type M3u8Downloader struct {
	base godown.AbstractDownloader
}

func (d *M3u8Downloader) Download(urlstr, dist string) {
	d.base.Init()
	tsdir := path.Join(path.Dir(dist), "ts")
	d.base.PrepareDist(tsdir)
	src, e := url.Parse(urlstr)
	if e != nil {
		panic(e)
	}
	m3u8File := d.base.FetchText(&http.Request{Method: http.MethodGet, URL: src})
	tsList := d.parseTsList(m3u8File)
	d.doDownload(tsList, urlstr, tsdir)
	d.combinTs(tsList, dist, tsdir)

}

func (d *M3u8Downloader) combinTs(tsList []string, dist, tsdir string) {
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	defer distFile.Close()
	if e != nil {
		panic(e)
	}
	for idx := range tsList {
		tssrc := path.Join(tsdir, strconv.Itoa(idx)+".ts")
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
	bar := pb.StartNow(len(tsList))
	for idx, tsurl := range tsList {
		ts, e := url.Parse(strings.Join([]string{base, tsurl}, "/"))
		if e != nil {
			panic(e)
		}
		tsDist := path.Join(tsdir, strconv.Itoa(idx)+".ts")
		d.base.HttpDown(&http.Request{Method: http.MethodGet, URL: ts}, tsDist)
		bar.Increment()
	}
}

func (d *M3u8Downloader) parseTsList(m3u8 string) []string {
	baseList := strings.Split(m3u8, "\n")
	distList := []string{}
	for _, line := range baseList {
		if strings.Contains(line, "#") {
			continue
		} else {
			distList = append(distList, line)
		}
	}
	return distList
}

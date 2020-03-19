package downloader

import (
	"net/http"
	"net/url"
)

type HttpDownloader struct {
	base godown.AbstractDownloader
}

func (d *HttpDownloader) Download(urlstr string, dist string) {
	d.base.Init()
	url, e := url.Parse(urlstr)
	if e != nil {
		panic(e)
	}
	request := http.Request{Method: "Get", URL: url}
	d.base.HttpDown(&request, dist)
}

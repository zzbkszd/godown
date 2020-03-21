package downloader

import (
	"github.com/zzbkszd/godown/godown"
	"github.com/zzbkszd/godown/godown/shadownet"
	"net/http"
	"net/url"
)

type HttpDownloader struct {
	base godown.AbstractDownloader
}

func (d *HttpDownloader) Download(urlstr string, dist string) {
	d.base.Client = shadownet.GetShadowClient()
	url, e := url.Parse(urlstr)
	if e != nil {
		panic(e)
	}
	request := http.Request{Method: "Get", URL: url}
	d.base.HttpDown(&request, dist)
}

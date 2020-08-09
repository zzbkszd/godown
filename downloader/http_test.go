package downloader

import (
	"fmt"
	"net/http"
	"testing"
)

func TestDebug(t *testing.T) {
	httpd := DefaultHttpDownloader
	murl := "https://cchzkj.cn/mediah5/cn/eyJ1c2VyX2lkIjo0MDcxNDgyNywibGFzdGxvZ2luIjoxNTk2OTM4ODMxfQ.bd1f05b7695a648a4ed6743af8131b79.647f26c4377ca995cb48b5bb31d2c5a0aed354cad496a2c53d3b2c68/1596942710/eslorflbkjwzig/irbjslsvjf/240/130649.m3u8"
	checkUrl := murl + "?check=true"
	checkResult, e := httpd.FetchText(QuickRequest(http.MethodGet, checkUrl, http.Header{}))
	if e != nil {
		panic(e)
	}
	fmt.Println(checkResult)
	header := http.Header{}
	//header.Add("Host", "cchzkj.cn")
	header.Add("Origin", "https://web.ruizhirongda.com")
	//header.Add("Referer", "https://web.ruizhirongda.com/?cache=9578519&license=550e633cc2d854782a1fb0277a16895d")
	m3u8file, e := httpd.FetchText(QuickRequest(http.MethodGet, murl, header))
	if e != nil {
		panic(e)
	}
	//meta := ParseM3u8(m3u8file)

	//fmt.Println(meta)
	fmt.Println(m3u8file)
}

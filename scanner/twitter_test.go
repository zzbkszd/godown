package scanner

import (
	"github.com/zzbkszd/godown/downloader"
	"github.com/zzbkszd/godown/godown/shadownet"
	"testing"
)

func TestTwitterUserScanner_scan(t *testing.T) {
	vd := downloader.HttpDownloader{}
	//vd.Client = shadownet.GetShadowClient(shadownet.LocalShadowConfig)
	vd.Client = shadownet.GetShadowClient(shadownet.LocalShadowConfig)

	scanner := &TwitterUserScanner{
		BaseScanner: BaseScanner{
			D: vd,
		},
		TwitterUserId: "DensTinon", // 输入为用户的screen_id，就是URL最后的那个，以及@的那个
		LastTweet:     "-1",
		Limit:         20,
	}
	e := scanner.querySinglePage("UniG19", "", "", true, 10)
	if e != nil {
		panic(e)
	}
	//text, e := vd.FetchText(downloader.QuickRequest(http.MethodGet, "https://free-proxy-list.net/", nil))
	//h := scanner.getRandomHeader()
	//fmt.Println(h)
	//text, e := vd.FetchText(downloader.QuickRequest(http.MethodGet, "https://twitter.com/UniG19", nil))
	//if e != nil {
	//	panic(e)
	//}
	//fmt.Println(text)
}

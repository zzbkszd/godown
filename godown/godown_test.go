package godown

import (
	"fmt"
	"github.com/zzbkszd/godown/downloader"
	"github.com/zzbkszd/godown/godown/shadownet"
	"net/http"
	"path"
	"testing"
)

func TestDownloadCollect(t *testing.T) {

	collect := &Collect{
		Name:        "Test Collect",
		Type:        TYPE_VIDEO,
		Description: "Collect for test",
		Cover:       "",
		Source: []string{
			"https://www.xvideos.com/video44476201/_",
			"https://www.xvideos.com/video35345593/_",
			"https://www.xvideos.com/video35382343/_~04",
			"https://www.xvideos.com/video28205543/_x_1",
			"https://www.xvideos.com/video38593919/tumblr_~_",
		},
	}

	ctx := Godown{
		DataPath: path.Join("..", "data"),
	}

	ctx.DownloadCollect(collect)
}

func TestTwitterCollect(t *testing.T) {
	collect, err := TwitterCollect("DensTinon", "", 20)
	if err != nil {
		panic(err)
	}
	fmt.Println("collect size:", collect.Size())
	fmt.Println(collect.Source)
	//ctx := Godown{
	//	DataPath: path.Join("I:", "godown"),
	//}
	//ctx.DownloadCollect(collect)
}

func TestDownProxy(t *testing.T) {
	client := shadownet.GetShadowClient(shadownet.LocalShadowConfig)
	h := downloader.HttpDownloader{}
	h.SetClient(client)
	urlStr := "https://the.earth.li/~sgtatham/putty/latest/w64/putty.zip"
	h.HttpDown(downloader.QuickRequest(http.MethodGet, urlStr, nil), "./putty.html")
}

package downloader

import (
	"fmt"
	"testing"
)

func TestVideoDownlaod(t *testing.T) {
	vd := VideoDonwloader{}
	vd.Download("https://www.bilibili.com/video/av83641887", "../../data/dist.flv")
}

func TestBilibili(t *testing.T) {
	vd := VideoDonwloader{}
	vd.Init()
	exts, err := vd.bilibiliExtractor("https://www.bilibili.com/video/av83641887")
	if err != nil {
		panic(err)
	}
	fmt.Println(exts)
}

package downloader

import (
	"fmt"
	"github.com/zzbkszd/godown/godown/shadownet"
	"path"
	"regexp"
	"testing"
)

func TestUrlCutter(t *testing.T) {
	name := GetUrlFileName("https://www.xvideos.com/video44476201/_")
	fmt.Println(name)
}

func TestRegexp(t *testing.T) {
	webpage :=
		`abc.ef/\es<`
	reg := regexp.MustCompile(`[\\/:*?\"<>|]`)
	res := reg.ReplaceAllString(webpage, `_`)
	fmt.Println(res)
}

func TestVideoDownlaod(t *testing.T) {
	vd := VideoDonwloader{}
	// 这个……下载速度……贼尼玛不科学！
	vd.Download("https://cn.pornhub.com/view_video.php?viewkey=ph5db26265db653",
		"../../data/dist.mp4")
}

func testExtractor(url string, extractor func(string, *AbstractDownloader) (*VideoInfo, error)) {
	vd := VideoDonwloader{}
	vd.Client = shadownet.GetShadowClient(shadownet.LocalShadowConfig)
	exts, err := extractor(url, &vd.AbstractDownloader)
	if err != nil {
		panic(err)
	}
	fmt.Println(exts)
}

func TestXvideos(t *testing.T) {
	testExtractor("https://www.xvideos.com/video4588838/biker_takes_his_girl", xvideosExtractor)
}

func TestBilibili(t *testing.T) {
	testExtractor("https://www.bilibili.com/video/av83641887", bilibiliExtractor)
}

func TestPornhub(t *testing.T) {
	testExtractor("https://cn.pornhub.com/view_video.php?viewkey=ph5db26265db653", pornhubExtractor)
}

func TestTwitterDownloader(t *testing.T) {
	td := TwitterDonwloader{}
	td.Client = shadownet.GetShadowClient(shadownet.LocalShadowConfig)
	//https://twitter.com/EfWMSxfSCrHY8v4/status/1242779629296840704 video
	// https://twitter.com/isisdna123/status/1243084980516843521 picture
	//https://twitter.com/i/web/status/1242779629296840704
	td.Download("https://twitter.com/i/web/status/1242779629296840704",
		path.Join("..", "data", "twitter"))
}

func TestEhentai(t *testing.T) {
	d := EhentaiDonwloader{}
	d.Client = shadownet.GetShadowClient(shadownet.LocalShadowConfig)
	d.DisplayProgress = true
	err := d.Download("https://e-hentai.org/g/1597700/21efbef24d/", "../../data/ehentai")
	if err != nil {
		panic(err)
	}
}

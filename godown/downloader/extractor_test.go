package downloader

import (
	"fmt"
	"regexp"
	"testing"
)

func TestUrlCutter(t *testing.T) {
	name := GetUrlFileName("https://www.xvideos.com/video44476201/_")
	fmt.Println(name)
}

func TestRegexp(t *testing.T) {
	webpage :=
		`html5player.setVideoTitle('Biker Takes his Girl');
	    html5player.setSponsors(false);
	    html5player.setVideoUrlLow('https://video-hw.xvideos-cdn.com/videos/3gp/8/7/1/xvideos.com_871d333c861767f358f727f293bb8d86-1.mp4?e=1584946580&ri=1024&rs=85&h=34c2e6f05aca47eed897045a98e5ce58');
	    html5player.setVideoUrlHigh('https://video-hw.xvideos-cdn.com/videos/mp4/8/7/1/xvideos.com_871d333c861767f358f727f293bb8d86-1.mp4?e=1584946580&ri=1024&rs=85&h=5781b1e46033920eed273c6b381df86e');
	    html5player.setVideoHLS('https://hls-hw.xvideos-cdn.com/videos/hls/87/1d/33/871d333c861767f358f727f293bb8d86-1/hls.m3u8?e=1584946580&l=0&h=3359c9fe319d1376330125135a7f6747');
	    html5player.setThumbUrl('https://img-hw.xvideos-cdn.com/videos/thumbslll/87/1d/33/871d333c861767f358f727f293bb8d86/871d333c861767f358f727f293bb8d86.2.jpg');
	    html5player.setThumbUrl169('https://img-hw.xvideos-cdn.com/videos/thumbs169lll/87/1d/33/871d333c861767f358f727f293bb8d86/871d333c861767f358f727f293bb8d86.1.jpg');
	    html5player.setRelated(video_related);`
	reg := regexp.MustCompile("setVideo([^(]+)\\([\"\\'](http.+?)[\"\\']")
	allmatch := reg.FindAllStringSubmatch(webpage, 3)
	for _, match := range allmatch {
		fmt.Printf("type: %s, url:%s \n", match[1], match[2])
	}
}

func TestVideoDownlaod(t *testing.T) {
	vd := VideoDonwloader{}
	// 这个……下载速度……贼尼玛不科学！
	vd.Download("https://cn.pornhub.com/view_video.php?viewkey=ph5db26265db653",
		"../../data/dist.mp4")
}

func testExtractor(url string, extractor func(string, *AbstractDownloader) (*VideoInfo, error)) {
	vd := VideoDonwloader{}
	vd.Init()
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

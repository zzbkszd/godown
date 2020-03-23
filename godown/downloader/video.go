package downloader

import (
	"crypto/md5"
	"fmt"
	"github.com/tidwall/gjson"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

/**
视频下载器
从视频网站下载指定视频。
go 版本的 youtube-dl 简单实现。
解析代码参照 youtube-dl的 extract 源码
*/

type VideoDonwloader struct {
	AbstractDownloader
	sourceUrl     string // 来源网站
	extract       func(url string) (VideoInfo, error)
	extractMapper map[string]func(url string) (VideoInfo, error)
}

func (vd *VideoDonwloader) Download(urlStr string, dist string) error {
	vd.Init()
	vd.mapExtractor()
	vd.sourceUrl = urlStr
	parsedUrl, _ := url.Parse(urlStr)
	host := parsedUrl.Hostname()

	if extrator, ok := vd.extractMapper[host]; ok {
		info, _ := extrator(urlStr)
		if info.infos[0].ext == "m3u8" {
			m3u8d := M3u8Downloader{}
			m3u8d.Client = vd.Client
			m3u8d.Download(info.infos[0].url, dist)
		} else {
			vd.HttpDown(quickRequest(http.MethodGet, info.infos[0].url, info.infos[0].headers), dist)
		}
	} else {
		return fmt.Errorf("Unsupport video source!")
	}
	return nil
}

func (vd *VideoDonwloader) mapExtractor() {
	vd.extractMapper = map[string]func(url string) (VideoInfo, error){
		"www.bilibili.com": vd.bilibiliExtractor,
	}
}

type ExtractInfo struct {
	url     string      // 视频下载链接
	ext     string      // 下载链接格式
	headers http.Header // 网络请求的header
	name    string      // 视频清晰度名称
}

func (ei *ExtractInfo) String() string {
	return fmt.Sprintf("{ url: %s , name: %s }", ei.url, ei.name)
}

type VideoInfo struct {
	id    string
	src   string
	name  string
	infos []ExtractInfo
}

/**
bilibili support
only video as https://www.bilibili.com/video/av83641887
*/
func (vd *VideoDonwloader) bilibiliExtractor(videoUrl string) (info VideoInfo, e error) {
	_APP_KEY := "iVGUTjsxvpLeuDCf"
	_BILIBILI_KEY := "aHRmhWMLkdeMuILqORnYZocwMBpMEOdt"
	info = VideoInfo{src: videoUrl}
	id := videoUrl[strings.LastIndex(videoUrl, "/")+3:]
	info.id = id
	webpage := vd.FetchText(quickRequest(http.MethodGet, videoUrl, nil))
	// 获取CID
	reg := regexp.MustCompile("\\bcid(?:[\"\\']:|=)(?P<cid>\\d+)")
	// 0 是全文，1是cid
	cid := reg.FindStringSubmatch(webpage)[1]

	// 调用接口获取信息
	eis := make([]ExtractInfo, 0)
	RENDITIONS := []string{"qn=80&quality=80&type=", "quality=2&type=mp4"}
	for _, rendition := range RENDITIONS {
		payload := fmt.Sprintf("appkey=%s&cid=%s&otype=json&%s", _APP_KEY, cid, rendition)
		sign := fmt.Sprintf("%x", (md5.Sum([]byte(payload + _BILIBILI_KEY))))
		apiCall := fmt.Sprintf("http://interface.bilibili.com/v2/playurl?%s&sign=%s", payload, sign)
		respJsonStr := vd.FetchText(quickRequest(http.MethodGet, apiCall, nil))
		if respJsonStr == "" || len(respJsonStr) < 200 {
			continue
		}
		fmt.Println(respJsonStr)
		downloadUrl := gjson.Get(respJsonStr, "durl*.0.url")
		format := gjson.Get(respJsonStr, "format")
		quality := gjson.Get(respJsonStr, "quality")
		hder := http.Header{}
		hder.Add("Referer", vd.sourceUrl)
		eis = append(eis, ExtractInfo{
			url:     downloadUrl.Str,
			ext:     format.Str,
			headers: hder,
			name:    quality.Str,
		})
		// 优先获取一个最优画质，然后就放弃治疗
		break
	}
	info.infos = eis
	return
}

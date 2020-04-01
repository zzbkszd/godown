package downloader

import (
	"context"
	"crypto/md5"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/robertkrimen/otto"
	"github.com/tidwall/gjson"
	"github.com/zzbkszd/godown/godown/common"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ffmpeg 视频编码器
var ffmpeg_codec = "h264_qsv" // 适用于intel核显的硬件加速

/**
视频下载器
从视频网站下载指定视频。
go 版本的 youtube-dl 简单实现。
解析代码参照 youtube-dl的 extract 源码
*/
var extractMapper = map[string]func(url string, downloader *AbstractDownloader) (*VideoInfo, error){
	"www.bilibili.com": bilibiliExtractor,
	"www.xvideos.com":  xvideosExtractor,
	"cn.pornhub.com":   pornhubExtractor,
	"www.pornhub.com":  pornhubExtractor,
}

type VideoDonwloader struct {
	AbstractDownloader
	sourceUrl string // 来源网站
	extract   func(url string, vd *AbstractDownloader) (*VideoInfo, error)
	AutoName  bool // 是否从目标网站自动读取文件名，若是则不使用指定文件名
	// 但是无论如何dist中都要指定一个默认文件名
}

func (vd *VideoDonwloader) Download(urlStr string, dist string) (realDist string, err error) {
	vd.Init()
	vd.sourceUrl = urlStr
	parsedUrl, _ := url.Parse(urlStr)
	host := parsedUrl.Hostname()

	if extrator, ok := extractMapper[host]; ok {
		// 只下载第一个，所以如果要做优选，则需要在解析器内进行排序
		info, err := extrator(urlStr, &vd.AbstractDownloader)
		if err != nil {
			panic(err)
			return "", nil
		}
		distExt := filepath.Ext(dist)
		distPath := dist
		if info.name != "" {
			videoName := FormatFilename(info.name) + distExt
			distPath = path.Join(filepath.Dir(dist), videoName)
		}
		realDist = distPath
		fmt.Println("[Video Downloader] download video ext: ", info.infos[0].ext)
		fmt.Println("[Video Downloader] download video dist ext: ", distExt)
		if info.infos[0].ext == "hls" {
			m3u8d := M3u8Downloader{
				Threads: 5,
				AbstractDownloader: AbstractDownloader{
					Client: vd.Client,
					CommonProgress: common.CommonProgress{
						DisplayProgress: vd.DisplayProgress,
					},
				},
			}
			m3u8d.Client = vd.Client
			if distExt != ".ts" {
				tempDist := path.Join(path.Dir(distPath),
					fmt.Sprintf("temp.%d.ts", int(time.Now().Unix())))
				m3u8d.Download(info.infos[0].url, tempDist)
				err := vd.videoTrans(tempDist, distPath)
				if err == nil {
					os.Remove(tempDist)
				} else {
					return "", nil
				}
			} else {
				m3u8d.Download(info.infos[0].url, distPath)
			}
		} else {
			httpd := MultipartHttpDownloader{headers: info.infos[0].headers}
			httpd.Client = vd.Client
			httpd.Download(info.infos[0].url, distPath)
		}
	} else {
		fmt.Println("[Video Downloader] Unsupport video source:", host)
		err = fmt.Errorf("Unsupport video source!")
		return
	}
	return
}

func (vd *VideoDonwloader) videoTrans(src string, dist string) error {
	absSrc, _ := filepath.Abs(src)
	absDist, _ := filepath.Abs(dist)
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", absSrc, "-c:v",
		ffmpeg_codec, "-c:a", "aac", absDist)
	cmd.Start()
	//done := ctx.Done()
	fmt.Println("[Video Downloader] video format decode start!")
	err := cmd.Wait()
	fmt.Println("[Video Downloader] video format decode done!")
	return err
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
todo support bangumi url as : https://www.bilibili.com/bangumi/play/ss32381
*/
func bilibiliExtractor(videoUrl string, vd *AbstractDownloader) (info *VideoInfo, e error) {
	_APP_KEY := "iVGUTjsxvpLeuDCf"
	_BILIBILI_KEY := "aHRmhWMLkdeMuILqORnYZocwMBpMEOdt"
	info = &VideoInfo{src: videoUrl}
	id := videoUrl[strings.LastIndex(videoUrl, "/")+3:]
	info.id = id
	webpage, e := vd.FetchText(quickRequest(http.MethodGet, videoUrl, nil))
	if e != nil {
		return nil, e
	}
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
		respJsonStr, e := vd.FetchText(quickRequest(http.MethodGet, apiCall, nil))
		if e != nil {
			return nil, e
		}
		if respJsonStr == "" || len(respJsonStr) < 200 {
			continue
		}
		downloadUrl := gjson.Get(respJsonStr, "durl*.0.url")
		format := gjson.Get(respJsonStr, "format")
		quality := gjson.Get(respJsonStr, "quality")
		hder := http.Header{}
		hder.Add("Referer", videoUrl)
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

/**
xvideos support
*/
func xvideosExtractor(videoUrl string, vd *AbstractDownloader) (info *VideoInfo, e error) {
	info = &VideoInfo{src: videoUrl}
	webpage, e := vd.FetchText(quickRequest(http.MethodGet, videoUrl, nil))
	if e != nil {
		return nil, e
	}
	reg := regexp.MustCompile("setVideo([^(]+)\\([\"\\'](http.+?)[\"\\']")
	allmatch := reg.FindAllStringSubmatch(webpage, 3)
	eis := make([]ExtractInfo, 0)
	hder := http.Header{}
	hder.Add("Referer", videoUrl)
	for _, match := range allmatch {
		vtype, vurl := match[1], match[2]
		ext := ""
		switch vtype {
		case "UrlLow":
			ext = "3gp"
			break
		case "UrlHigh":
			ext = "mp4"
			break
		case "HLS": //  hls需要先下载hls列表，再下载hls文件
			hlsPage, e := vd.FetchText(quickRequest(http.MethodGet, vurl, nil))
			if e != nil {
				continue
			}
			baseList := strings.Split(hlsPage, "\n")
			// 可能有多个，这里只拿第一个
			// warning: 默认第一个的清晰度是最高的，如果不是可能会有bug
			for _, line := range baseList {
				if len(line) == 0 || strings.HasPrefix(line, "#") {
					continue
				} else {
					vurl = getParentUrl(vurl) + "/" + line
					ext = "hls"
					break
				}
			}
			break
		}
		if ext == "3gp" || ext == "mp4" {
			continue // 宁可不要这破烂3GP
		}
		eis = append(eis, ExtractInfo{
			url:     vurl,
			ext:     ext,
			headers: hder,
			name:    vtype,
		})
	}
	info.infos = eis
	if len(eis) != 0 { // 找到了就算了，别报错了
		e = nil
	}
	return
}

/**
pornhub support
thanks for https://github.com/treant5612/pornhub-dl
*/
func pornhubExtractor(videoUrl string, vd *AbstractDownloader) (info *VideoInfo, e error) {
	info = &VideoInfo{src: videoUrl}
	getWebpage := func(plat string) (r string, e error) {
		header := http.Header{}
		header.Set("Cookie", fmt.Sprintf("platform=%s;", plat))
		r, e = vd.FetchText(quickRequest(http.MethodGet, videoUrl, header))
		return
	}
	webpage, e := getWebpage("pc")
	if e != nil {
		return nil, e
	}
	document, _ := goquery.NewDocumentFromReader(strings.NewReader(webpage))
	if title, ok := document.Find("[property='og:title']").Attr("content"); ok {
		info.name = title
	}
	playerDiv := document.Find("#player")
	if id, ok := playerDiv.Attr("data-video-id"); ok {
		info.id = id
	} else {
		return nil, fmt.Errorf("Parse pornhub video id error")
	}
	scriptDiv := playerDiv.Find("script")
	scripts := scriptDiv.Text()
	script := strings.Split(scripts, "loadScriptUniqueId")[0]

	vm := otto.New()
	_, e = vm.Run(script)
	// 可能有错，但是不影响获取flashvars_
	//if e != nil {
	//	fmt.Println("DEBUG: otto run bug! script:")
	//	fmt.Println(script)
	//	return
	//}
	value, e := vm.Get("flashvars_" + info.id)
	object, e := value.Export()
	objMapper := object.(map[string]interface{})

	mediaDefined := objMapper["mediaDefinitions"].([]map[string]interface{})
	eis := make([]ExtractInfo, 0)
	for _, v := range mediaDefined {
		quality, ok := v["quality"].(string)
		if !ok {
			continue
		}
		ext := v["format"].(string)
		//if ext == "mp4" { // 只下载m3u8
		//	continue
		//}
		vurl := v["videoUrl"].(string)
		if ext == "hls" {
			master, err := vd.FetchText(quickRequest(http.MethodGet, vurl, nil))
			if err != nil {
				return nil, err
			}
			baseList := strings.Split(master, "\n")
			// 可能有多个，这里只拿第一个
			for _, line := range baseList {
				if len(line) == 0 || strings.HasPrefix(line, "#") {
					continue
				} else {
					vurl = getParentUrl(vurl) + "/" + line
					ext = "hls"
					break
				}
			}
		}
		eis = append(eis, ExtractInfo{
			url:     vurl,
			ext:     ext,
			headers: nil,
			name:    quality,
		})
	}
	info.infos = eis
	return
}

func getMeta(doc goquery.Document, selector, attrName string) string {
	if attr, ok := doc.Find("[property='og:title']").Attr("content"); ok {
		return attr
	} else {
		return ""
	}
}

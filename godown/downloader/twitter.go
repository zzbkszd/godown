package downloader

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

/**
twitter 下载器
基于twitter的特性，它的下载是下载多个文件
以twitter_id.info文件作为数据引导的入口
即为数据-附件的形式。
类似的可以用于新浪微博之类
*/
type TwitterDonwloader struct {
	AbstractDownloader
	guestToken string
}

type TweetInfo struct {
	Id        string
	Text      string
	Uploader  string
	Timestamp time.Time
	Picture   []string
	Video     []string
}

func (td *TwitterDonwloader) Download(urlStr string, dist string) error {
	info, e := td.twitterExtractor(urlStr)
	if e != nil {
		return e
	}
	for idx, u := range info.Picture {
		mname := info.Id + "." + GetUrlFileName(u)
		md := path.Join(dist, mname)
		td.HttpDown(quickRequest(http.MethodGet, u, nil), md)
		info.Picture[idx] = mname
	}
	for idx, u := range info.Video {
		mname := info.Id + "." + GetUrlFileName(u)
		if strings.HasSuffix(mname, "m3u8") {
			m3u8d := M3u8Downloader{}
			m3u8d.Client = td.Client
			mname = mname + ".ts"
			md := path.Join(dist, mname)
			m3u8d.Download(u, md)
		} else {
			md := path.Join(dist, mname)
			httpd := HttpDownloader{}
			httpd.Client = td.Client
			httpd.Download(u, md)
		}
		info.Video[idx] = mname
	}
	infoDist := path.Join(dist, fmt.Sprintf("%s.json", info.Id))
	infoJson, e := json.Marshal(info)
	if e != nil {
		return e
	}
	return ioutil.WriteFile(infoDist, infoJson, 0777)
}

func (td *TwitterDonwloader) callTwitterApi(path, videoId string, query map[string]string) (string, error) {
	API_BASE := "https://api.twitter.com/"
	headers := http.Header{}
	headers.Add("Authorization", "Bearer AAAAAAAAAAAAAAAAAAAAAPYXBAAAAAAACLXUNDekMxqa8h%2F40K4moUkGsoc%3DTYfbDKbT3jJPCEVnMYqilB28NHfOPqkca3qaAxGfsyKCs0wRbw")
	if td.guestToken == "" {
		guestReq, err := http.NewRequest(http.MethodGet, API_BASE+"1.1/guest/activate.json", nil)
		if err != nil {
			return "", err
		}
		guestReq.Header = headers
		guestJson, err := td.FetchText(guestReq)
		if err != nil {
			return "", err
		}
		guestToken := gjson.Get(guestJson, "guest_token")
		td.guestToken = guestToken.String()
	}
	headers.Add("x-guest-token", td.guestToken)
	values := url.Values{}
	for k, v := range query {
		values.Add(k, v)
	}
	apiUrl := API_BASE + path + "?" + values.Encode()
	apiReq, err := http.NewRequest(http.MethodGet, apiUrl, nil)
	if err != nil {
		return "", err
	}
	apiReq.Header = headers

	respJson, err := td.FetchText(apiReq)

	return respJson, err
}

func parseTwitteInfo(info string) *TweetInfo {
	id := gjson.Get(info, "id")
	text := gjson.Get(info, "full_text")
	uploader := gjson.Get(info, "user.screen_name")
	//date := gjson.Get(info, "created_at")
	pics := make([]string, 0)
	video := make([]string, 0)

	medias := gjson.Get(info, "extended_entities.media")
	for _, m := range medias.Array() {
		minfo := m.Map()
		mtype := minfo["type"]
		switch mtype.String() {
		case "photo":
			pics = append(pics, minfo["media_url"].String())
		case "video":
			vinfo := minfo["video_info"]
			vlist := vinfo.Map()["variants"].Array()
			// 此处最好按照bitrate排序取最大，目前是根据排列取最后一个，依赖于数据特性
			last := vlist[len(vlist)-1]
			vurl := last.Map()["url"].String()
			video = append(video, vurl)

		}
	}

	return &TweetInfo{
		Id:        id.String(),
		Text:      text.String(),
		Uploader:  uploader.String(),
		Timestamp: time.Time{},
		Picture:   pics,
		Video:     video,
	}
}

func (td *TwitterDonwloader) twitterExtractor(tweetUrl string) (*TweetInfo, error) {
	reg := regexp.MustCompile(`https?://(?:(?:www|m(?:obile)?)\.)?twitter\.com/(?:(?:i/web|[^/]+)/status|statuses)/(?P<id>\d+)`)
	tweetId := reg.FindStringSubmatch(tweetUrl)[1]
	api := "1.1/statuses/show/%s.json"
	//api := "2/timeline/conversation/%s.json" // v2版接口已经上线，不知道v1.1还能支持多长时间
	status, err := td.callTwitterApi(fmt.Sprintf(api, tweetId),
		tweetId, map[string]string{
			"cards_platform":                 "Web-12",
			"include_cards":                  "1",
			"include_can_media_tag":          "1",
			"include_ext_media_availability": "true",
			"ext":                            "mediaStats",
			"tweet_mode":                     "extended",
		})
	//tweetInfo := parseTwitteInfo(gjson.Get(status, "globalObjects.tweets."+tweetId).String())
	tweetInfo := parseTwitteInfo(status)
	return tweetInfo, err
}

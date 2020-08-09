package extractor

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/zzbkszd/godown/downloader"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
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
	downloader.AbstractDownloader
	api *twitterApi
}

type TweetInfo struct {
	Id        string
	Text      string
	Uploader  string
	Timestamp time.Time
	Picture   []string
	Video     []string
}

/**
api环境主要维护一个guestToken， 所以有一个全局的就可以了
*/
var commonTwitterApi = &twitterApi{}

func (td *TwitterDonwloader) Download(urlStr string, dist string) (string, error) {
	td.api = &twitterApi{}
	td.api.init(td.Client)
	info, e := td.api.twitterExtractor(urlStr)
	if e != nil {
		return "", e
	}
	for idx, u := range info.Picture {
		mname := info.Id + "." + downloader.GetUrlFileName(u)
		md := path.Join(dist, mname)
		td.HttpDown(downloader.quickRequest(http.MethodGet, u, nil), md)
		info.Picture[idx] = mname
	}
	for idx, u := range info.Video {
		mname := info.Id + "." + downloader.GetUrlFileName(u)
		if strings.HasSuffix(mname, "m3u8") {
			m3u8d := downloader.M3u8Downloader{}
			m3u8d.Client = td.Client
			mname = mname + ".ts"
			md := path.Join(dist, mname)
			m3u8d.Download(u, md)
		} else {
			md := path.Join(dist, mname)
			httpd := downloader.HttpDownloader{}
			httpd.Client = td.Client
			httpd.Download(u, md)
		}
		info.Video[idx] = mname
	}
	infoDist := path.Join(dist, fmt.Sprintf("%s.json", info.Id))
	infoJson, e := json.Marshal(info)
	if e != nil {
		return "", e
	}
	e = ioutil.WriteFile(infoDist, infoJson, 0777)
	return infoDist, e
}

type twitterApi struct {
	downloader.AbstractDownloader
	gustTokenMu sync.Mutex
	guestToken  string
}

/**
todo 需要考虑到重复下载的时候，不要每次都初始化一个twitterApi
*/
func (td *twitterApi) callTwitterApi(path, videoId string, query map[string]string) (string, error) {
	API_BASE := "https://api.twitter.com/"
	headers := http.Header{}
	headers.Add("Authorization", "Bearer AAAAAAAAAAAAAAAAAAAAAPYXBAAAAAAACLXUNDekMxqa8h%2F40K4moUkGsoc%3DTYfbDKbT3jJPCEVnMYqilB28NHfOPqkca3qaAxGfsyKCs0wRbw")
	getToken := func() (string, error) {
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
		return guestToken.String(), nil
	}
	if td.guestToken == "" {
		td.gustTokenMu.Lock()
		// double check！
		if td.guestToken == "" {
			if token, err := getToken(); err == nil {
				td.guestToken = token
			} else {
				return "", err
			}
		}
		td.gustTokenMu.Unlock()
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

func (td *twitterApi) twitterExtractor(tweetUrl string) (*TweetInfo, error) {
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

func (td *twitterApi) init(client *http.Client) {
	if td.Client == nil {
		td.Client = client
	}
}

/**
读取timeline
twitter的apiv2将整体的格式进行了统一，分为数据表，用户表和timeline三个部分，不论是列表还是单个都是一样
*/
func (td *TwitterDonwloader) timelineExtractor(twitter string) {
	//https://api.twitter.com/graphql/P8ph10GzBbdMqWZxulqCfA/UserByScreenName?variables=%7B%22screen_name%22%3A%22realdonaldtrump%22%2C%22withHighlightedLabel%22%3Atrue%7D
	//https://api.twitter.com/2/timeline/profile/25073877.json?include_profile_interstitial_type=1&include_blocking=1&include_blocked_by=1&include_followed_by=1&include_want_retweets=1&include_mute_edge=1&include_can_dm=1&include_can_media_tag=1&skip_status=1&cards_platform=Web-12&include_cards=1&include_composer_source=true&include_ext_alt_text=true&include_reply_count=1&tweet_mode=extended&include_entities=true&include_user_entities=true&include_ext_media_color=true&include_ext_media_availability=true&send_error_codes=true&simple_quoted_tweets=true&include_tweet_replies=false&userId=25073877&count=20&cursor=HBaAwLS1u9SevyIAAA%3D%3D&ext=mediaStats%2ChighlightedLabel%2CcameraMoment
	//https://api.twitter.com/2/timeline/profile/#userid#.json
	// todo  对于如何通过UID来查询还存在问题

}

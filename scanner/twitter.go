package scanner

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"github.com/zzbkszd/godown/downloader"
	"github.com/zzbkszd/godown/godown/extractor"
	"github.com/zzbkszd/godown/godown/shadownet"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type TwitterUserScanner struct {
	BaseScanner
	TwitterUserId string // 输入为用户的screen_id，就是URL最后的那个，以及@的那个
	LastTweet     string
	Limit         int
}

type Tweet struct {
	ScreenName    string
	UserName      string
	UserId        string
	TweetId       string
	TweetUrl      string
	Timestamp     int64
	Text          string
	TextHtml      string
	Links         string
	HashTags      string
	HasMedia      bool
	ImgUrls       []string
	VideoUrl      string
	Likes         int
	IsReplyTo     bool
	ParentTweetId string
}

func (s *TwitterUserScanner) Scan() (*ScannerResult, error) {
	afterPart := "include_available_features=1&include_entities=1&include_new_items_bar=true"
	timelineUrl := fmt.Sprintf("https://twitter.com/i/profiles/show/%s/timeline/tweets?%s",
		s.TwitterUserId, afterPart)
	headers := http.Header{}
	headers.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	headers.Add("Referer", fmt.Sprintf("https://twitter.com/%s", s.TwitterUserId))
	headers.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8")
	headers.Add("X-Twitter-Active-User", "yes")
	headers.Add("X-Requested-With", "XMLHttpRequest")
	headers.Add("Accept-Language", "en-US")
	client := shadownet.GetShadowClient(shadownet.LocalShadowConfig)

	getPage := func(lastId string) (tweets []string, hasMore bool, err error) {
		tweets = make([]string, 0)
		qurl := timelineUrl
		if lastId != "" {
			qurl += fmt.Sprintf("&max_position=%s", lastId)
		}
		timelineReq, err := http.NewRequest(http.MethodGet, timelineUrl, nil)
		if err != nil {
			return nil, false, err
		}
		timelineReq.Header = headers
		resp, err := client.Do(timelineReq)
		if err != nil {
			return nil, false, err
		}
		page, err := ioutil.ReadAll(resp.Body)
		pageStr := string(page)
		hasMore = gjson.Get(pageStr, "has_more_items").Bool()
		htmlStr := gjson.Get(pageStr, "items_html").String()
		document, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
		if err != nil {
			return nil, hasMore, err
		}
		document.Find(".stream-item").Each(
			func(idx int, selection *goquery.Selection) {
				if tweetId, ok := selection.Attr("data-item-id"); ok {
					tweets = append(tweets,
						fmt.Sprintf("https://www.twitter.com/%s/status/%s", s.TwitterUserId, tweetId))
				}
			})
		return
	}
	tweetList := make([]*DataSource, 0)
	lastId := s.LastTweet
	for {
		tweets, hasMore, err := getPage(lastId)
		if err != nil {
			return nil, err
		}
		for _, t := range tweets {
			tweetList = append(tweetList, &DataSource{Uri: t})
		}
		if !hasMore || (s.Limit > 0 && len(tweetList) >= s.Limit) {
			break
		}
	}
	return &ScannerResult{SourceList: tweetList, Downloader: &extractor.TwitterDonwloader{}}, nil
}

func (s *TwitterUserScanner) getQueryUrl(query, lang, pos string, fromUser bool) string {
	INIT_URL_USER := "https://twitter.com/%s"
	INIT_URL := "https://twitter.com/search?f=tweets&vertical=default&q=%s&l=%s"
	RELOAD_URL := "https://twitter.com/i/search/timeline?f=tweets&vertical=" +
		"default&include_available_features=1&include_entities=1&" +
		"reset_error_state=false&src=typd&max_position=%s&q=%s&l=%s"
	RELOAD_URL_USER := "https://twitter.com/i/profiles/show/%s/timeline/tweets?" +
		"include_available_features=1&include_entities=1&" +
		"max_position=%s&reset_error_state=false"
	if fromUser {
		if pos == "" {
			return fmt.Sprintf(INIT_URL_USER, query)
		} else {
			return fmt.Sprintf(RELOAD_URL_USER, query, pos)
		}
	}
	if pos == "" {
		return fmt.Sprintf(INIT_URL, query, lang)
	} else {
		return fmt.Sprintf(RELOAD_URL, query, pos, lang)
	}
}

func (s *TwitterUserScanner) getRandomHeader(query string) http.Header {
	HEADERS_LIST := []string{
		"Mozilla/5.0 (Windows; U; Windows NT 6.1; x64; fr; rv:1.9.2.13) Gecko/20101203 Firebird/3.6.13",
		"Mozilla/5.0 (compatible, MSIE 11, Windows NT 6.3; Trident/7.0; rv:11.0) like Gecko",
		"Mozilla/5.0 (Windows; U; Windows NT 6.1; rv:2.2) Gecko/20110201",
		"Opera/9.80 (X11; Linux i686; Ubuntu/14.10) Presto/2.12.388 Version/12.16",
		"Mozilla/5.0 (Windows NT 5.2; RW; rv:7.0a1) Gecko/20091211 SeaMonkey/9.23a1pre",
	}
	rand.Seed(time.Now().Unix())
	rnd := rand.Intn(5)
	headers := http.Header{}
	headers.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	//headers.Add("Referer", fmt.Sprintf("https://twitter.com/%s", query))
	headers.Add("X-Twitter-Active-User", "yes")
	headers.Add("Accept-Language", "en-US")
	headers.Add("User-Agent", HEADERS_LIST[rnd])
	headers.Add("X-Requested-With", "XMLHttpRequest")
	return headers
}

func (s *TwitterUserScanner) querySinglePage(query, lang, pos string, fromUser bool, retry int) error {
	url := s.getQueryUrl(query, lang, pos, fromUser)
	fmt.Println(url)
	response, e := s.BaseScanner.D.FetchText(
		downloader.QuickRequest(http.MethodGet, url, s.getRandomHeader(query)))
	if e != nil {
		return e
	}
	//fmt.Println(response)
	html := ""
	if pos == "" {
		html = response
	} else {
		html = ""
		html = gjson.Get(response, "items_html").String()
	}
	_, e = fromHtml(html)
	if e != nil {
		return e
	}
	return nil

}

func fromHtml(html string) (*Tweet, error) {
	doc, e := goquery.NewDocumentFromReader(strings.NewReader(html))
	if e != nil {
		return nil, e
	}
	tweets := doc.Find("li")
	fmt.Println(tweets)
	return nil, nil

}

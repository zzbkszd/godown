package scanner

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"github.com/zzbkszd/godown/godown/extractor"
	"github.com/zzbkszd/godown/godown/shadownet"
	"io/ioutil"
	"net/http"
	"strings"
)

type TwitterUserScanner struct {
	BaseScanner
	TwitterUserId string // 输入为用户的screen_id，就是URL最后的那个，以及@的那个
	LastTweet     string
	Limit         int
}

func (s *TwitterUserScanner) scan() (*ScannerResult, error) {
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

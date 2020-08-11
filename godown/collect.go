package godown

import (
	"bufio"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
	"github.com/zzbkszd/godown/downloader"
	"github.com/zzbkszd/godown/godown/shadownet"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

/**
数据集和下载任务
*/
var TYPE_VIDEO = 1
var TYPE_NOVEL = 2
var TYPE_TWITTER = 3

type DownloadTask struct {
	Dist       string
	Source     string
	Type       int
	Downloader downloader.Downloader
}

type Collect struct {
	Name        string   // 集合名称
	Type        int      // 集合类型
	Description string   // 集合描述
	Cover       string   // 集合封面（可选）
	Source      []string // 集合下载列表
}

func (c *Collect) Size() interface{} {
	return len(c.Source)
}

/**
从列表文件中生成Collect
文件中每行为一个URL，集合名称为文件名，描述为空
type 需要用户指定， 一个集合中只能包含一个类型
*/
func ListFileCollect(file string, ctype, skip int) (*Collect, error) {
	listFile, err := os.OpenFile(file, os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	finfo, err := listFile.Stat()
	if err != nil {
		return nil, err
	}
	fname := finfo.Name()
	bufReader := bufio.NewReader(listFile)
	source := []string{}
	for {
		line, _, e := bufReader.ReadLine()
		if e != nil {
			break
		}
		src := strings.Trim(string(line), " ")
		if strings.HasPrefix(src, "http") {
			source = append(source, string(line))
		}
	}
	return &Collect{
		Name:        fname,
		Type:        ctype,
		Description: "",
		Cover:       "",
		Source:      source[skip:],
	}, nil

}

/**
从twitter用户的timeline生成集合
输入为用户的screen_id，就是URL最后的那个，以及@的那个
*/
func TwitterCollect(user, last string, limit int) (*Collect, error) {
	afterPart := "include_available_features=1&include_entities=1&include_new_items_bar=true"
	timelineUrl := fmt.Sprintf("https://twitter.com/i/profiles/show/%s/timeline/tweets?%s",
		user, afterPart)
	headers := http.Header{}
	headers.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	headers.Add("Referer", fmt.Sprintf("https://twitter.com/%s", user))
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
						fmt.Sprintf("https://www.twitter.com/%s/status/%s", user, tweetId))
				}
			})
		return
	}
	tweetList := []string{}
	lastId := last
	for {
		tweets, hasMore, err := getPage(lastId)
		if err != nil {
			return nil, err
		}
		tweetList = append(tweetList, tweets...)
		if !hasMore || (limit > 0 && len(tweetList) >= limit) {
			break
		}
	}
	return &Collect{
		Name:        user,
		Type:        TYPE_TWITTER,
		Description: "",
		Cover:       "",
		Source:      tweetList,
	}, nil

}

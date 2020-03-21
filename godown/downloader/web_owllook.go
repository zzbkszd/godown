package downloader

import (
	"bufio"
	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
	"github.com/zzbkszd/godown/godown"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

// https://owllook.net 小说网站， 从目录页开始下载
// 注意该网站就他娘的可以用http爬取，服务端对于https的跨域支持有问题
// 并发抓取会导致请求被拦截，所以目前只用单线程慢慢爬
type OwllookDonwloader struct {
	base     godown.AbstractDownloader
	chapters []string
	names    []string
}

func (d *OwllookDonwloader) Download(urlstr, dist string) {
	d.base.Init()
	url, e := url.Parse(urlstr)
	if e != nil {
		panic(e)
	}
	request := &http.Request{Method: "Get", URL: url}
	html := d.base.FetchText(request)
	d.chapters, d.names = d.parseChapters(html)
	chapterCount := len(d.chapters)
	bar := pb.StartNew(chapterCount)
	distFile, err := os.OpenFile(dist, os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	for idx, _ := range d.chapters {
		//d.downloadGo(idx, path.Join(path.Dir(dist), "chapter"), bar)
		d.downloadChapter(d.chapters[idx], d.names[idx], distFile)
		bar.Increment()
	}
	bar.Finish()

	//d.combinChpater(chapterCount, dist)

	/*
		https://www.owllook.net/chapter?url=http://www.mangg.com/id7769/&novels_name=%E8%AF%A1%E7%A7%98%E4%B9%8B%E4%B8%BB
		https://www.owllook.net/owllook_content?url=http://www.mangg.com//id7769/4389310.html&name=%E7%AC%AC%E4%B8%80%E7%AB%A0%20%E7%BB%AF%E7%BA%A2&chapter_url=http://www.mangg.com/id7769/&novels_name=%E8%AF%A1%E7%A7%98%E4%B9%8B%E4%B8%BB
	*/
}

// 用于多线程下载的预备方法
func (d *OwllookDonwloader) downloadGo(idx int, dist string,
	bar *pb.ProgressBar) {
	distPath := path.Join(dist, strconv.Itoa(idx))
	d.base.PrepareDist(distPath)
	distFile, e := os.OpenFile(distPath, os.O_CREATE, 0777)
	if e != nil {
		panic(e)
	}
	d.downloadChapter(d.chapters[idx], d.names[idx], distFile)
	bar.Increment()
}

func (d *OwllookDonwloader) downloadChapter(chapter, name string, distFile *os.File) {
	chapter_url, e := url.Parse(chapter)
	chapter_html := d.base.FetchText(&http.Request{Method: "Get", URL: chapter_url})
	cd, e := goquery.NewDocumentFromReader(strings.NewReader(chapter_html))
	if e != nil {
		panic(e)
	}
	content, e := cd.Find("#content").First().Html()
	content = strings.ReplaceAll(content, "<br/>", "\n")
	if len(content) < 100 {
		panic("content too short!" + chapter_html)
	}

	formated_content := ""
	reader := bufio.NewReader(strings.NewReader(content))
	line, _, err := reader.ReadLine()
	for err == nil {
		formated_content += strings.TrimLeft(string(line), " \ufeff")
		formated_content += "\n"
		line, _, err = reader.ReadLine()
	}

	_, err = distFile.WriteString("\n" + name + "\n\n")
	_, err = distFile.WriteString(formated_content)
	if err != nil {
		panic(err)
	}
}

func (d *OwllookDonwloader) parseChapters(html string) ([]string, []string) {
	document, e := goquery.NewDocumentFromReader(strings.NewReader(html))
	if e != nil {
		panic(e)
	}
	list := document.Find("#list a")
	content_url, _ := document.Find("#content_url").First().Attr("value")
	chapter_url, _ := document.Find("#url").First().Attr("value")
	novels_name, _ := document.Find("#novels_name").First().Attr("value")
	if content_url == "1" {
		content_url = ""
	}
	chapters := []string{}
	names := []string{}
	list.Each(func(idx int, selection *goquery.Selection) {
		href, _ := selection.Attr("href")
		name := selection.Text()
		chapter := "http://www.owllook.shadownet/owllook_content?url=" + content_url + href +
			"&name=" + url.QueryEscape(name) + "&chapter_url=" + chapter_url +
			"&novels_name=" + url.QueryEscape(novels_name)
		chapters = append(chapters, chapter)
		names = append(names, name)
	})
	return chapters, names
}

func (d *OwllookDonwloader) combinChpater(cnt int, dist string) {
	distFile, e := os.OpenFile(dist, os.O_CREATE, 0777)
	defer distFile.Close()
	if e != nil {
		panic(e)
	}
	for i := 0; i < cnt; i++ {
		tssrc := path.Join(path.Dir(dist), "chapter", strconv.Itoa(i))
		tsFile, e := os.OpenFile(tssrc, os.O_RDONLY, 0777)
		defer tsFile.Close()
		if e != nil {
			panic(e)
		}
		io.Copy(distFile, tsFile)
	}
}
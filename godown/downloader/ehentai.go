package downloader

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/zzbkszd/godown/godown/common"
	"net/http"
	"path"
	"strconv"
	"strings"
)

/**
e-hentai下载器
输入：目录页url
输出：在目标目录输出所有图片
*/
type EhentaiDonwloader struct {
	AbstractDownloader
	api *twitterApi
}

func (ed *EhentaiDonwloader) Download(src, dist string) (string, error) {
	list, err := ed.ehentaiExtractor(src)
	if err != nil {
		return "", err
	}
	tasks := []func() error{}
	ed.InitProgress(int64(len(list)), false)
	for page, l := range list {
		lp := page
		ll := l
		tasks = append(tasks, func() error {
			pic, err := ed.ehentaiParsePicture(ll)
			if err != nil {
				return err
			}
			pname := strconv.Itoa(lp) + "_" + GetUrlFileName(pic)
			err = ed.HttpDown(quickRequest(http.MethodGet, pic, nil), path.Join(dist, pname))
			ed.UpdateProgress(1)
			return err
		})
	}
	cycle := &common.MultiTaskCycle{
		Threads: 5,
	}
	err = cycle.CostTasks(tasks)
	ed.CloseProgress()
	return dist, err

}

func (ed *EhentaiDonwloader) ehentaiExtractor(listUrl string) ([]string, error) {
	listPage, err := ed.FetchText(quickRequest(http.MethodGet, listUrl, nil))
	if err != nil {
		return nil, err
	}
	listDoc, err := goquery.NewDocumentFromReader(strings.NewReader(listPage))
	if err != nil {
		return nil, err
	}
	ptt := listDoc.Find(".ptt td")
	totalPages := []string{}
	for i := 0; i < ptt.Size()-2; i++ {
		pages, err := ed.ehentaiSingleLst(fmt.Sprintf("%s?p=%d", listUrl, i))
		if err != nil {
			return nil, err
		}
		totalPages = append(totalPages, pages...)
	}
	return totalPages, err
}

func (ed *EhentaiDonwloader) ehentaiSingleLst(listUrl string) ([]string, error) {
	listPage, err := ed.FetchText(quickRequest(http.MethodGet, listUrl, nil))
	if err != nil {
		return nil, err
	}
	listDoc, err := goquery.NewDocumentFromReader(strings.NewReader(listPage))
	if err != nil {
		return nil, err
	}
	gdtm := listDoc.Find(".gdtm a")
	pages := []string{}
	gdtm.Each(func(idx int, selection *goquery.Selection) {
		if purl, exist := selection.Attr("href"); exist {
			pages = append(pages, purl)
		}
	})
	return pages, err
}

func (ed *EhentaiDonwloader) ehentaiParsePicture(src string) (string, error) {
	picPage, err := ed.FetchText(quickRequest(http.MethodGet, src, nil))
	if err != nil {
		return "", err
	}
	picDoc, err := goquery.NewDocumentFromReader(strings.NewReader(picPage))
	if err != nil {
		return "", err
	}
	if pic, exist := picDoc.Find("#img").Attr("src"); exist {
		return pic, nil
	} else {
		return "", fmt.Errorf("Picture not found")
	}
}

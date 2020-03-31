package downloader

import (
	"github.com/zzbkszd/godown/godown/shadownet"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

/** **************************
Some useful utils
************************** **/
// simple and typical http request
func quickRequest(method string, urlStr string, headers http.Header) (req *http.Request) {
	if headers == nil {
		headers = shadownet.DefaultHeader
	} else {
		headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36")
	}
	req, _ = http.NewRequest(http.MethodGet, urlStr, nil)
	req.Header = headers
	return req
}

func getParentUrl(base string) string {
	parent := strings.Split(base, "/")
	return strings.Join(parent[:len(parent)-1], "/")
}

func GetUrlFileName(base string) string {
	if strings.HasPrefix(base, "http") {
		if u, e := url.Parse(base); e == nil {
			path := strings.Split(u.Path, "/")
			return path[len(path)-1]
		}
	}
	s, e := strings.LastIndex(base, "/"), strings.Index(base, "?")
	if s == -1 {
		s = 0
	}
	if e > 0 {
		return base[s:e]
	} else {
		return base[s:]
	}

}

func FormatFilename(name string) (formated string) {
	reg := regexp.MustCompile(`[\\/:*?\"<>|]`)
	return reg.ReplaceAllString(name, `_`)
}

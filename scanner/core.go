package scanner

import (
	"github.com/zzbkszd/godown/downloader"
	"net/http"
)

type ScannerCore interface {
	scan() (*ScannerResult, error)
}

type BaseScanner struct {
	D downloader.HttpDownloader
}

type DataSource struct {
	Uri        string
	HttpHeader http.Header
}

type ScannerResult struct {
	Downloader downloader.Downloader
	SourceList []*DataSource
}

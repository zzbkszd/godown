package downloader

import (
	common2 "github.com/zzbkszd/godown/common"
	"testing"
)

func TestM3u8_Download(t *testing.T) {
	md := M3u8Downloader{
		AbstractDownloader: AbstractDownloader{
			CommonProgress: common2.CommonProgress{
				DisplayProgress: false,
				DisplayOnUpdate: true,
			},
		},
	}
	tsurl := "https://cchzkj.cn/mediah5/cn/eyJ1c2VyX2lkIjo0MDcxNDgyNywibGFzdGxvZ2luIjoxNTk2OTM4ODMxfQ.bd1f05b7695a648a4ed6743af8131b79.647f26c4377ca995cb48b5bb31d2c5a0aed354cad496a2c53d3b2c68/1596942710/eslorflbkjwzig/irbjslsvjf/240/130649.m3u8"
	md.Download(tsurl,
		"I:/godown/tests.mp4")
}

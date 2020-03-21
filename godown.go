package main

import (
	"github.com/zzbkszd/godown/godown"
	"github.com/zzbkszd/godown/godown/downloader"
	"path"
)

func downloaderTest(down godown.Downloader, src, dist string) {
	down.Download(src, dist)
}

func main() {
	downloader := downloader.HttpDownloader{}
	//downloader := downloader.OwllookDonwloader{}
	test_file := "https://github.com/Anuken/Mindustry/releases/download/v104.1/server-release.jar"
	//test_m3u8 := "http://youku.com-www-163.com/20180519/3432_620c9a63/1000k/hls/index.m3u8"
	//test_owl := "http://www.owllook.net/chapter?url=http://www.mangg.com/id7769/&novels_name=%E8%AF%A1%E7%A7%98%E4%B9%8B%E4%B8%BB"
	//test_owl := "http://www.owllook.net"
	distPath := path.Join(".", "data", "dist.txt")
	downloaderTest(&downloader, test_file, distPath)
}

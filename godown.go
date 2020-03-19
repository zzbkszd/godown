package godown

import (
	"./godown"
	"path"
)
import "./godown/downloader"

func downloaderTest(down godown.Downloader, src, dist string) {
	down.Download(src, dist)
}

func main() {
	//httpDownloader := downloader.HttpDownloader{}
	m3u8Downloader := downloader.M3u8Downloader{}
	//test_file := "https://github.com/Anuken/Mindustry/releases/download/v104.1/server-release.jar"
	test_m3u8 := "http://youku.com-www-163.com/20180519/3432_620c9a63/1000k/hls/index.m3u8"
	distPath := path.Join(".", "data", "dist.mp4")
	downloaderTest(&m3u8Downloader, test_m3u8, distPath)
}

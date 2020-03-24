package main

import (
	"github.com/zzbkszd/godown/godown/downloader"
	"path"
)

func main() {
	downloader := downloader.VideoDonwloader{}
	dist := path.Join(".", "data", "dist.mp4")
	downloader.Download("https://www.xvideos.com/video35180883/_", dist)
}

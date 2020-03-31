package main

import (
	"github.com/zzbkszd/godown/godown"
	"path"
)

func main() {
	collect, err := godown.ListFileCollect("data/日产.dat", godown.TYPE_VIDEO, 16)
	if err != nil {
		panic(err)
	}
	ctx := godown.Godown{
		DataPath:           path.Join("..", "data"),
		WorkThreads:        1,
		DefaultVideoFormat: "ts",
	}

	ctx.DownloadCollect(collect)
}

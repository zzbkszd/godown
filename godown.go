package main

import (
	"github.com/zzbkszd/godown/godown"
	"path"
)

func main() {
	//shadownet.GetShadowPool()
	collect := &godown.Collect{
		Name:        "bilibili",
		Type:        godown.TYPE_VIDEO,
		Description: "Collect for test bv",
		Cover:       "",
		Source: []string{
			"https://www.bilibili.com/video/BV1F741117vi",
		},
	}

	ctx := godown.Godown{
		DataPath: path.Join("..", "data"),
	}

	ctx.DownloadCollect(collect)
}

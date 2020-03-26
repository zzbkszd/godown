package godown

import (
	"path"
	"testing"
)

func TestDownloadCollect(t *testing.T) {

	collect := &Collect{
		Name:        "Test Collect",
		Type:        TYPE_VIDEO,
		Description: "Collect for test",
		Cover:       "",
		Source: []string{
			"https://www.xvideos.com/video44476201/_",
			"https://www.xvideos.com/video35345593/_",
			"https://www.xvideos.com/video35382343/_~04",
			"https://www.xvideos.com/video28205543/_x_1",
			"https://www.xvideos.com/video38593919/tumblr_~_",
		},
	}

	ctx := Godown{
		DataPath: path.Join("..", "data"),
	}

	ctx.DownloadCollect(collect)
}

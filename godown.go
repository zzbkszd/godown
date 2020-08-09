package main

import (
	"context"
	"fmt"
	"github.com/zzbkszd/godown/godown"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

/**
批量格式转换
*/
func transVideoAll(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		fmt.Println(path)
		ext := filepath.Ext(path)
		dist := strings.ReplaceAll(path, ext, ".mp4")
		fmt.Println(dist)
		videoTrans(path, dist)
		os.Remove(path)
		return nil
	})

}
func videoTrans(src string, dist string) error {
	absSrc, _ := filepath.Abs(src)
	absDist, _ := filepath.Abs(dist)
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", absSrc, "-c:v",
		"copy", "-c:a", "copy", absDist)
	cmd.Start()
	err := cmd.Wait()
	return err
}

func downloadFileCollect(file string) {
	collect, err := godown.ListFileCollect(file, godown.TYPE_VIDEO, 5)
	if err != nil {
		panic(err)
	}
	ctx := godown.Godown{
		DataPath:           path.Join("..", "data", "0507data"),
		WorkThreads:        1,
		DefaultVideoFormat: "mp4",
	}

	ctx.DownloadCollect(collect)
}

func main() {
	//transVideoAll(`../data/collect/jump.dat`)
	downloadFileCollect("data/0507/歌迪斯")
}

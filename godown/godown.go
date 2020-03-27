package godown

import (
	"fmt"
	"github.com/zzbkszd/godown/godown/common"
	"github.com/zzbkszd/godown/godown/downloader"
	"github.com/zzbkszd/godown/godown/shadownet"
	"os"
	"path"
)

/**
Godown的后端上下文
*/
type Godown struct {
	DataPath string // 数据保存位置
}

/**
下载一个集合
*/
func (god *Godown) DownloadCollect(collect *Collect) error {
	collectBase := path.Join(god.DataPath, "collcet", collect.Name)
	pg := &common.CommonProgress{
		DisplayProgress: true,
	}
	os.MkdirAll(collectBase, 0777)
	pg.InitProgress(int64(len(collect.Source)), false)
	tasks := []func() error{}
	client := shadownet.GetShadowClient(shadownet.LocalShadowConfig)
	for idx, task := range collect.Source {
		var downer downloader.Downloader
		name := downloader.GetUrlFileName(task)
		switch collect.Type {
		case TYPE_VIDEO:
			downer = &downloader.VideoDonwloader{}
			downer.SetClient(client)
			name = fmt.Sprintf("%d.mp4", idx)
		case TYPE_TWITTER:
			downer = &downloader.TwitterDonwloader{}
			downer.SetClient(client)
		}
		ltask := task
		tasks = append(tasks, func() error {
			err := downer.Download(ltask, path.Join(collectBase, name))
			pg.UpdateProgress(1)
			return err
		})
	}
	cycle := common.MultiTaskCycle{
		Threads:   5,
		TryOnFail: false,
	}
	return cycle.CostTasks(tasks)
}

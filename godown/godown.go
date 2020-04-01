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
	DataPath           string // 数据保存位置
	WorkThreads        int    // 工作线程数量
	DefaultVideoFormat string
}

/**
下载一个集合
*/
func (god *Godown) DownloadCollect(collect *Collect) error {
	fmt.Println("[GoDown] initial collect downloader")
	collectBase := path.Join(god.DataPath, "collect", collect.Name)
	pg := &common.CommonProgress{
		DisplayProgress: false,
		DisplayOnUpdate: true,
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
			downer = &downloader.VideoDonwloader{
				AutoName: true,
				AbstractDownloader: downloader.AbstractDownloader{
					Client: client,
					CommonProgress: common.CommonProgress{
						DisplayProgress: true,
					},
				},
			}
			name = fmt.Sprintf("%d.%s", idx, god.DefaultVideoFormat)
		case TYPE_TWITTER:
			downer = &downloader.TwitterDonwloader{}
			downer.SetClient(client)
		}
		ltask := task
		tasks = append(tasks, func() error {
			_, err := downer.Download(ltask, path.Join(collectBase, name))
			pg.UpdateProgress(1)
			return err
		})
	}
	if god.WorkThreads == 0 {
		god.WorkThreads = 5
	}
	fmt.Printf("[GoDown] downloader initialed, %d tasks and %d work threads", len(tasks), god.WorkThreads)
	cycle := common.MultiTaskCycle{
		Threads:   god.WorkThreads,
		TryOnFail: true,
	}
	return cycle.CostTasks(tasks)
}

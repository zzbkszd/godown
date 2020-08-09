package godown

import (
	"fmt"
	common2 "github.com/zzbkszd/godown/common"
	downloader2 "github.com/zzbkszd/godown/downloader"
	"github.com/zzbkszd/godown/godown/extractor"
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
下载一个集合，下载完成后就结束
*/
func (god *Godown) DownloadCollect(collect *Collect) error {
	fmt.Println("[GoDown] initial collect extractor")
	collectBase := path.Join(god.DataPath, "collect", collect.Name)
	pg := &common2.CommonProgress{
		DisplayProgress: false,
		DisplayOnUpdate: true,
	}
	os.MkdirAll(collectBase, 0777)
	pg.InitProgress(int64(len(collect.Source)), false)
	tasks := []func() error{}
	//client := shadownet.GetShadowClient(&shadownet.ShadowConfig{
	//	Ip:           "198.255.78.36",
	//	Port:         8099,
	//	Password:     "eIW0Dnk69454e6nSwuspv9DmS201tQ0D",
	//	CryptoMethod: "aes-256-cfb",
	//})
	//client := shadownet.GetShadowClient(shadownet.LocalShadowConfig)
	client := shadownet.GetLocalClient()
	for idx, task := range collect.Source {
		var downer downloader2.Downloader
		name := downloader2.GetUrlFileName(task)
		switch collect.Type {
		case TYPE_VIDEO:
			downer = &extractor.VideoDonwloader{
				AutoName: true,
				AbstractDownloader: downloader2.AbstractDownloader{
					Client: client,
					CommonProgress: common2.CommonProgress{
						DisplayProgress: true,
					},
				},
			}
			name = fmt.Sprintf("%d.%s", idx, god.DefaultVideoFormat)
		case TYPE_TWITTER:
			downer = &extractor.TwitterDonwloader{}
			downer.SetClient(client)
		}

		ltask := task
		tasks = append(tasks, func() error {
			_, err := downer.Download(ltask, path.Join(collectBase, name))
			if err == nil {
				pg.UpdateProgress(1)
			}
			return err
		})
	}
	if god.WorkThreads == 0 {
		god.WorkThreads = 5
	}
	fmt.Printf("[GoDown] extractor initialed, %d tasks and %d work threads\n", len(tasks), god.WorkThreads)
	cycle := common2.MultiTaskCycle{
		Threads:   god.WorkThreads,
		TryOnFail: true,
	}
	return cycle.CostTasks(tasks)
}

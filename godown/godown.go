package godown

import (
	"fmt"
	"github.com/zzbkszd/godown/godown/downloader"
	"net/http"
	"os"
	"path"
	"sync"
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
	os.MkdirAll(collectBase, 0777)
	dt := downloadThread{}
	putCh := dt.GetPutCh()
	go dt.Run()
	for idx, task := range collect.Source {
		var downer downloader.Downloader
		name := downloader.GetUrlFileName(task)
		switch collect.Type {
		case TYPE_VIDEO:
			downer = &downloader.VideoDonwloader{
				AbstractDownloader: downloader.AbstractDownloader{
					//Client: shadownet.GetShadowClient(shadownet.LocalShadowConfig),
					Client: http.DefaultClient,
				},
			}
			name = fmt.Sprintf("%d.mp4", idx)
		}

		//fmt.Println("push download task dist name:", name)
		putCh <- &DownloadTask{
			Dist:       path.Join(collectBase, name),
			Source:     task,
			Type:       collect.Type,
			Downloader: downer,
		}
	}
	dt.StopUntilDone()
	return nil
}

type errTask struct {
	task *DownloadTask
	err  error
}
type downloadThread struct {
	putCh   chan *DownloadTask
	errCh   chan errTask
	stopCh  chan int
	stopMu  *sync.Mutex
	curTask *DownloadTask
}

func (dt *downloadThread) GetStopCh() chan int {
	if dt.stopCh == nil {
		dt.stopCh = make(chan int)
	}
	return dt.stopCh
}
func (dt *downloadThread) GetPutCh() chan *DownloadTask {
	if dt.putCh == nil {
		dt.putCh = make(chan *DownloadTask, 3) // 这是一个有缓冲的管道
	}
	return dt.putCh
}
func (dt *downloadThread) GetErrCh() chan errTask {
	if dt.errCh == nil {
		dt.errCh = make(chan errTask)
	}
	return dt.errCh
}

/**
等待当前任务完成后再结束
这是一个阻塞方法
??? select 本身不会阻塞么？
!!! select 会阻塞，但是外层的主程序不会啊。这个方法可以令主程序等待
*/
func (dt *downloadThread) StopUntilDone() {
	dt.GetStopCh() <- 1
	dt.stopMu.Lock()
	dt.stopMu.Unlock()
}

/**
下载循环
*/
func (dt *downloadThread) Run() {
	if dt.putCh == nil {
		dt.GetPutCh()
	}
	if dt.stopMu == nil {
		dt.stopMu = &sync.Mutex{}
	}
	for {
		select {
		case <-dt.stopCh: // 优先关闭，即使任务管道中还有数据
			break
		case t := <-dt.putCh:
			dt.stopMu.Lock() // 锁死，可以通过锁状态查看下载是否完成
			dt.curTask = t
			err := t.Downloader.Download(t.Source, t.Dist)
			if dt.errCh != nil {
				dt.errCh <- errTask{task: t, err: err}
			}
			dt.curTask = nil
			dt.stopMu.Unlock()
		}
	}
}

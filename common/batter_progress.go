package common

import (
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"sync"
)

type ProgressInfo struct {
	done   int64
	total  int64
	isByte bool
}
type ProgressAble interface {
	ProgressChan() chan *ProgressInfo
	GetProgress() *ProgressInfo
}

/**
进度管理的抽象实现
*/
type CommonProgress struct {
	DisplayProgress bool               // 是否在终端打印进度信息（pb)
	DisplayOnUpdate bool               // 是否在更新时打印进度比例（done/total）
	progressChan    chan *ProgressInfo // 进度回调
	progressInfo    *ProgressInfo      // 当前的进度信息
	progressMutex   *sync.Mutex        // 更新进度信息的互斥锁
	pbbar           *pb.ProgressBar    // 终端显示的进度条
}

func (d *CommonProgress) SetDisplay(isSet bool) {
	d.DisplayProgress = isSet
}

func (d *CommonProgress) ProgressChan() chan *ProgressInfo {
	d.progressMutex.Lock()
	if d.progressChan != nil {
		d.progressChan = make(chan *ProgressInfo)
	}
	d.progressMutex.Unlock()
	return d.progressChan
}

func (d *CommonProgress) GetProgress() *ProgressInfo {
	return d.progressInfo
}

func (d *CommonProgress) InitProgress(total int64, isByte bool) {
	d.progressMutex = &sync.Mutex{}
	d.progressMutex.Lock()
	d.progressInfo = &ProgressInfo{
		done:   0,
		total:  total,
		isByte: isByte,
	}
	if d.DisplayProgress {
		d.pbbar = pb.New64(total)
		d.pbbar.Start()
		if isByte {
			d.pbbar.Set(pb.Bytes, true)
		}
	}
	d.progressMutex.Unlock()
}

func (d *CommonProgress) UpdateProgress(p int64) {
	d.progressMutex.Lock()
	d.progressInfo.done += p
	if d.DisplayProgress {
		d.pbbar.Add64(p)
	}
	if d.DisplayOnUpdate {
		fmt.Printf("\n current progress: %d/%d\n", d.progressInfo.done, d.progressInfo.total)
	}
	if d.progressChan != nil {
		d.progressChan <- d.progressInfo
	}
	d.progressMutex.Unlock()
}

func (d *CommonProgress) CloseProgress() {
	if d.DisplayProgress {
		d.pbbar.Finish()
	}
	if d.progressChan != nil {
		close(d.progressChan)
	}
}

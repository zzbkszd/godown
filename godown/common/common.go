package common

import (
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
	progressChan    chan *ProgressInfo // 进度回调
	progressInfo    *ProgressInfo      // 当前的进度信息
	progressMutex   *sync.Mutex        // 更新进度信息的互斥锁
	pbbar           *pb.ProgressBar    // 终端显示的进度条
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

/**
多线程任务循环
*/
type MultiTaskCycle struct {
	taskCh  chan func() error
	doneCh  chan int
	Threads int

	useWg bool
	wg    *sync.WaitGroup

	TryOnFail bool // 失败后是否重试
}

func (m *MultiTaskCycle) CostTasks(tasks []func() error) error {
	m.useWg = true
	m.wg = &sync.WaitGroup{}
	m.wg.Add(len(tasks))
	m.Startup()
	for _, t := range tasks {
		m.PushTask(t)
	}
	m.wg.Wait()
	return nil
}

func (m *MultiTaskCycle) PushTask(task func() error) {
	m.taskCh <- task
}

func (m *MultiTaskCycle) Startup() {
	m.initCh()
	for i := 0; i < m.Threads; i++ {
		go m.workGo()
	}
}

func (m *MultiTaskCycle) Stop() {
	for i := 0; i < m.Threads; i++ {
		m.doneCh <- 1
	}
}

func (m *MultiTaskCycle) workGo() {
	for {
		select {
		case ts := <-m.taskCh:
			err := ts()
			if err != nil && m.TryOnFail {
				m.taskCh <- ts
				continue
			}
			if m.useWg {
				m.wg.Done()
			}
		case <-m.doneCh:
			break
		}
	}
}

func (m *MultiTaskCycle) initCh() {
	m.doneCh = make(chan int, m.Threads)
	m.taskCh = make(chan func() error, m.Threads)
}

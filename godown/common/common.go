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

/**
多线程任务集合，一个没什么卵用的玩意
*/
type TaskSet []func() error

func (set TaskSet) Add(task func() error) []func() error {
	return append(set, task)
}

/**
多任务并发循环模板代码
采用生产-消费者模式
可选的，在任务执行报错的时候是否进行重试（TryOnFail)
支持指定数量的消费者协程并发处理任务列表
两种模式：
- 一次输入整个任务列表，阻塞至处理完成 （CostTasks)
- 开启一个循环，并向队列输入任务，输入完成后手动结束循环。(PushTask)
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
				fmt.Println("[CYCELE DEBUG] task run error:" + err.Error())
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
	m.taskCh = make(chan func() error, m.Threads*100)
}

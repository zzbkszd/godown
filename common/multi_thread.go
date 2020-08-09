package common

import (
	"fmt"
	"sync"
)

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

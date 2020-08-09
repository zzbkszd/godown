package common

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestCycel(t *testing.T) {
	cycle := MultiTaskCycle{
		Threads:   5,
		TryOnFail: true,
	}
	putc := make(chan int, 50)
	cycle.Startup()
	wg := sync.WaitGroup{}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		cycle.PushTask(func() error {
			n := rand.Intn(10000)
			time.Sleep(time.Millisecond * 50)
			putc <- n
			wg.Done()
			return nil
		})
	}
	go func() {
		wg.Wait()
		close(putc)
	}()
	cnt := 0
	for n := range putc {
		fmt.Printf("%d - %d\n", cnt, n)
		time.Sleep(300 * time.Millisecond)
		cnt += 1
	}

}

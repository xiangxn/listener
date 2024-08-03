package tools

import (
	"fmt"
	"sync"

	"github.com/elliotchance/pie/v2"
	"github.com/xiangxn/go-multicall"
)

func ConcurrentMulticall(mc *multicall.Caller, calls []*multicall.Call, chunkLength int, maxConcurrent int) (results []*multicall.Call, err error) {
	type task struct {
		Index int
		Mcs   []*multicall.Call
		Err   error
	}
	var wg sync.WaitGroup
	chunk := pie.Chunk(calls, chunkLength)
	taskChan := make(chan task)
	concurrent := make(chan struct{}, maxConcurrent)
	for i, arr := range chunk {
		concurrent <- struct{}{}
		wg.Add(1)
		go func(muc *multicall.Caller, cs []*multicall.Call, index int) {
			defer wg.Done()
			result, err := muc.Call(nil, cs...)
			t := task{Index: index, Mcs: result, Err: err}
			if err != nil {
				fmt.Printf("ConcurrentMulticall [%d] error: %s \n", index, err.Error())
			}
			taskChan <- t
			<-concurrent
		}(mc, arr, i)
	}
	go func() {
		wg.Wait()
		close(taskChan)
	}()
	var tasks []task
	for result := range taskChan {
		tasks = append(tasks, result)
	}
	tasks = pie.SortUsing(tasks, func(a, b task) bool { return a.Index < b.Index })
	for _, t := range tasks {
		results = append(results, t.Mcs...)
		if t.Err != nil {
			err = t.Err // 暂时只保留最后一个错误
		}
	}
	return
}

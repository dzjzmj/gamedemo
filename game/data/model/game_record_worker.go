package model

import (
	"context"
	"sync"
)

type Worker struct {
	dataSize int //  == len(worker.data)
	dataChan chan interface{}
	data     []interface{} //
	f        func([]interface{})
}

// 单个任务结构体
// dataSize 单次处理的数量量
// f 处理数据的逻辑
func NewWorker(dataSize int, f func([]interface{})) *Worker {

	var w Worker

	w.dataSize = dataSize //
	w.data = make([]interface{}, 0, dataSize)
	w.dataChan = make(chan interface{}, 1000)
	w.f = f

	return &w
}

//func (w *Worker) SetFunc(f func([]interface{})) {
//	w.f = f
//}

func (w *Worker) Do() {
	w.f(w.data)
	w.data = make([]interface{}, 0, w.dataSize)
}

func (w *Worker) Push(v interface{}) {
	w.dataChan <- v
}

func (w *Worker) run(ctx context.Context) {

	for {

		select {
		case <-ctx.Done():

			return

		case v := <-w.dataChan:

			w.data = append(w.data, v)
			if len(w.data) >= w.dataSize {
				w.Do()
			}
		}
	}
}

type Workers struct {
	sync.WaitGroup
	count   int // 计数器
	size    int
	workers []*Worker
	cancel  context.CancelFunc
}

// workerNum 任务并行数
// workerDataSize 任务数据数据累计数
// f 执行方案
func NewWorkers(workerNum, workerDataSize int, f func([]interface{})) *Workers {

	var ctx, cancel = context.WithCancel(context.Background())

	var workers Workers
	workers.size = workerNum
	workers.workers = make([]*Worker, 0, workerNum)
	workers.cancel = cancel

	for i := 0; i < workerNum; i++ {

		workers.WaitGroup.Add(1)

		var worker = NewWorker(workerDataSize, f)

		go func() {
			worker.run(ctx)
			workers.WaitGroup.Done()
		}()

		workers.workers = append(workers.workers, worker)
	}

	return &workers
}

func (workers *Workers) Push(v interface{}) {

	if workers.count >= workers.size {
		workers.count = 0
	}

	workers.workers[workers.count].Push(v)
	workers.count++
}

func (workers *Workers) Done() {

	workers.cancel()
	workers.WaitGroup.Wait()

	for _, v := range workers.workers {
		v.Do()
	}
}

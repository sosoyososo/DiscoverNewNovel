package AsynWorker

import (
	"time"
)

/*
主要功能:
	异步执行任务
细节说明:
	1. 加入的任务自动开始执行。
	2. 可以限定使用的线程数量。
	3. 一个任务执行结束后，自动开始执行下一个，如果所有任务执行完毕，线程自动退出。
	4. 加入任务后判断当前使用的线程数量，如果不够最大数目，就使用新的线程执行任务，否则等待空闲线程执行任务
*/

/*
Task 代表需要执行的任务
*/
type Task interface {
	Action()
}

/*
DefaultTask 默认的 task 支持，默认啥都不做
*/
type DefaultTask struct {
	action func()
}

/*
Action 默认的实现
*/
func (t DefaultTask) Action() {
	if t.action != nil {
		t.action()
	}
}

/*
AsynWorker 代表一个可以使用 RoutineCount 个线程执行 Action 的管理器
*/
type AsynWorker struct {
	MaxRoutineCount    int
	RoutineWaitTimeOut time.Duration
	StopedActopn       func()
	taskQueue          chan Task
	runningCount       int
}

/*
New 使用默认值创建一个 Worker
*/
func New() AsynWorker {
	w := AsynWorker{}
	w.taskQueue = make(chan Task, 200)
	w.MaxRoutineCount = 5
	w.RoutineWaitTimeOut = 3
	w.runningCount = 0
	w.StopedActopn = func() {
	}
	return w
}

/*
AddHandlerTask 一个操作直接作为任务加入
*/
func (w AsynWorker) AddHandlerTask(hanlder func()) {
	task := DefaultTask{}
	task.action = hanlder
	w.taskQueue <- task

	if w.runningCount < w.MaxRoutineCount {
		w.runningCount++
		go w.actWithTimeout()
	}
}

/*
AddTask 新增一个任务
*/
func (w AsynWorker) AddTask(t Task) {
	w.taskQueue <- t

	if w.runningCount < w.MaxRoutineCount {
		w.runningCount++
		go w.actWithTimeout()
	}
}

/*
IsRuning 是否正在运行
*/
func (w *AsynWorker) IsRuning() bool {
	return w.runningCount > 0
}

/*
RemoveUnexcutedTasks 已经开始的工作不会被打断，但还没执行的工作会被抛弃
*/
func (w *AsynWorker) RemoveUnexcutedTasks() {
	select {
	case <-w.taskQueue:
	default:
	}
}

var count = 1

func (w *AsynWorker) actWithTimeout() {
	select {
	case t := <-w.taskQueue:
		t.Action()
		w.actWithTimeout()
	default:
		time.Sleep(w.RoutineWaitTimeOut * time.Second)
		w.act()
	}
	if w.runningCount <= 0 {
		w.StopedActopn()
	}
}

func (w *AsynWorker) act() {
	select {
	case t := <-w.taskQueue:
		t.Action()
		w.actWithTimeout()
	default:
		w.runningCount--
	}
	if w.runningCount <= 0 {
		w.StopedActopn()
	}
}

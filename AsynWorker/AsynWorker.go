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
	NOTE: 任务放大效应
	任务放大效应是指一个任务的执行导致多个任务的创建，这样会导致任务队列快速被用完。这时候，正在执行的线程，创建新的任务，插入任务队列，跟任务队列无法插入之间产生矛盾，有可能导致死锁。

	暂时想到的最优的解决方案: 使用可以自动扩任务队列
	优点：解决上述问题
	缺点：
		1. 任务队列的扩充是否要限制，不然的可能会导致系统资源被耗尽
		2. 增加内部复杂度，容易出现更多问题
	当前的做法：
		采用这个方案，不对资源进行限制，先实现再说
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
	StopedAction       func()

	runningCount    int
	maxTaskInQueue  int
	taskQueue       chan Task
	taskQueueBuffer []chan Task
	addLock         chan int
	reduceLock      chan int
}

/*
New 使用默认值创建一个 Worker
*/
func New() AsynWorker {
	w := AsynWorker{}
	w.MaxRoutineCount = 3
	w.RoutineWaitTimeOut = 10
	w.StopedAction = func() {
	}
	w.runningCount = 0
	w.maxTaskInQueue = 1024
	w.taskQueue = make(chan Task, w.maxTaskInQueue)
	w.taskQueueBuffer = []chan Task{}
	w.addLock = make(chan int, 1)
	w.reduceLock = make(chan int, 1)
	w.addLock <- 1
	w.reduceLock <- 1

	return w
}

/*
AddHandlerTask 一个操作直接作为任务加入
*/
func (w *AsynWorker) AddHandlerTask(hanlder func()) {
	task := DefaultTask{}
	task.action = hanlder
	w.AddTask(task)
}

/*
AddTask 新增一个任务
*/
func (w *AsynWorker) AddTask(t Task) {
	<-w.addLock
	select {
	case w.taskQueue <- t:
	default:
		for i := 0; i < len(w.taskQueueBuffer); i++ {
			if len(w.taskQueueBuffer[i]) < w.maxTaskInQueue {
				queue := w.taskQueueBuffer[i]
				queue <- t
				break
			}
		}
	}
	w.addLock <- 1

	if w.runningCount < w.MaxRoutineCount {
		w.runningCount++
		go w.actWithTimeout()
	}
}

func (w *AsynWorker) getTask() *Task {
	<-w.reduceLock
	select {
	case t := <-w.taskQueue:
		w.reduceLock <- 1
		return &t
	default:
		for i := 0; i < len(w.taskQueueBuffer); i++ {
			if len(w.taskQueueBuffer[i]) < w.maxTaskInQueue {
				queue := w.taskQueueBuffer[i]
				if len(queue) > 0 {
					t := <-queue
					w.reduceLock <- 1
					return &t
				}
			}
		}
	}
	w.reduceLock <- 1
	return nil
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
	t := w.getTask()
	if t == nil {
		time.Sleep(w.RoutineWaitTimeOut * time.Second)
		w.act()
	} else {
		(*t).Action()
		w.actWithTimeout()
	}
}

func (w *AsynWorker) act() {
	t := w.getTask()
	if t == nil {
		w.runningCount--
	} else {
		(*t).Action()
		w.actWithTimeout()
	}
	if w.runningCount <= 0 {
		w.StopedAction()
	}
}

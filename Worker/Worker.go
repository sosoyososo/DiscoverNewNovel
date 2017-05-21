package Worker

import (
	"fmt"
)

/*
Task 代表需要执行的任务
*/
type Task interface {
	Action()
}

type emptyTask struct{}

func (t emptyTask) Action() {}

/*
Worker 代表一个可以使用 RoutineCount 个线程执行 Action 的管理器
*/
type Worker struct {
	RoutineCount  int
	FinishAction  func()
	ch            chan Task
	stopedRoutine chan int
	isrunning     bool
	isStoped      bool
	nomoreTasks   bool
	index         int
}

/*
New 使用默认值创建一个 Worker
*/
func New() Worker {
	w := Worker{}
	w.ch = make(chan Task, 200)
	w.stopedRoutine = make(chan int, 10)
	w.isrunning = false
	w.isStoped = false
	w.RoutineCount = 5
	w.index = 0

	w.FinishAction = func() {
		fmt.Println("worker finished")
	}
	return w
}

/*
AddTask 新增一个任务
*/
func (w Worker) AddTask(t Task) {
	w.ch <- t
}

/*
Stop 停止执行，已经开始的工作不会被打断，但还没执行的工作会被抛弃
*/
func (w *Worker) Stop() {
	w.isStoped = true
}

/*
Finish 不会结束所有任务，只是表示任务已经提交结束了
*/
func (w *Worker) Finish() {
	w.nomoreTasks = true
	// 加入一系列空任务避免有线程还在获取任务
	for i := 0; i < w.RoutineCount; i++ {
		w.ch <- emptyTask{}
	}
}

/*
Run  没有启动就先启动指定个数的线程等待输入，执行操作
*/
func (w *Worker) Run() {
	if w.isrunning == false {
		w.isrunning = true
		for i := 0; i < w.RoutineCount; i++ {
			go w.act()
		}

		go func() {
			for i := 0; i < w.RoutineCount; i++ {
				<-w.stopedRoutine
			}

			if w.FinishAction != nil {
				w.FinishAction()
			}
		}()
	}
}

func (w *Worker) act() {
	//如果没有关闭，就获取内容，执行下次操作
	if w.isStoped == false && w.nomoreTasks == false {
		t := <-w.ch
		index := w.index
		w.index = index + 1
		fmt.Printf("第 %d 个任务开始\n", index)
		t.Action()
		fmt.Printf("第 %d 个任务结束\n", index)
		w.act()
	} else {
		w.stopedRoutine <- 1
	}
}

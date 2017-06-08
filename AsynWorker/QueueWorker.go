package AsynWorker

import "time"

/*
SynWorker 单线程串行执行任务
*/
type SynWorker struct {
	actionChanBuffer  []chan func()
	dbActionInRunning bool
	shouldEnd         bool
	addActionLock     chan int
	reduceActionLock  chan int
}

/*
AddAction 增加一个任务
*/
func (s *SynWorker) AddAction(action func()) {
	if s.shouldEnd == true {
		return
	}
	if s.addActionLock == nil {
		s.addActionLock = make(chan int, 1)
		s.addActionLock <- 1
	}
	<-s.addActionLock
	if s.actionChanBuffer == nil {
		actionQueue := make(chan func(), 1024)
		s.actionChanBuffer = []chan func(){actionQueue}
	}
	for i := 0; i < len(s.actionChanBuffer); i++ {
		queue := s.actionChanBuffer[i]
		if len(queue) < 1024 {
			queue <- action
			break
		} else if i == len(s.actionChanBuffer)-1 {
			actionQueue := make(chan func(), 1024)
			actionQueue <- action
			s.actionChanBuffer = append(s.actionChanBuffer, actionQueue)
			break
		}
	}
	s.addActionLock <- 1
	if s.dbActionInRunning == false {
		s.dbActionInRunning = true
		go s.realAction()
	}
}

/*
Stop 停止添加任务，队列中任务结束后退出执行
*/
func (s *SynWorker) Stop() {
	s.shouldEnd = true
}

func (s *SynWorker) realAction() {
	if s.reduceActionLock == nil {
		s.reduceActionLock = make(chan int, 1)
		s.reduceActionLock <- 1
	}
	<-s.reduceActionLock
	for i := 0; i < len(s.actionChanBuffer); i++ {
		queue := s.actionChanBuffer[i]
		if len(queue) > 0 {
			action := <-queue
			s.reduceActionLock <- 1
			action()
			s.realAction()
			return
		}
	}
	s.reduceActionLock <- 1
	if s.shouldEnd {
		s.dbActionInRunning = false
		return
	}
	time.Sleep(time.Second)
	s.realAction()
}

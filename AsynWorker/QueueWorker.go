package AsynWorker

import (
	"time"
)

/*
SynWorker 单线程串行执行任务
*/
type SynWorker struct {
	actionChan        chan func()
	dbActionInRunning bool
	shouldEnd         bool
}

/*
AddAction 增加一个任务
*/
func (s *SynWorker) AddAction(action func()) {
	if s.shouldEnd == true {
		return
	}
	if s.actionChan == nil {
		s.actionChan = make(chan func(), 1024)
	}
	s.actionChan <- action
	if s.dbActionInRunning == false {
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
	if s.shouldEnd {
		return
	}

	select {
	case action := <-s.actionChan:
		action()
		s.realAction()
	default:
		time.Sleep(time.Second)
	}

}

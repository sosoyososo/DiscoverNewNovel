package Discover

import (
	"fmt"

	"../AsynWorker"
	"../HtmlWorker"
	"github.com/PuerkitoBio/goquery"
)

/*
Worker 封装一个简单的html爬虫，使用一个url作为入口，扩散开来进行连接的遍历。
	1. 异步的获取新的连接
	2. 同一次启动不会遍历相同的连接
	3. 外部使用者来判断一个连接是否应该向下遍历
	4. 外部使用者对连接进行有效性判断
*/
type Worker struct {
	taskQueue AsynWorker.AsynWorker

	visitedURLList []string
	listLock       chan int

	runningCount int
	OnFinish     func()

	shouldContinueOnURL func(string) bool
	configHTMLWorker    func(*HtmlWorker.Worker)
	urlConvert          func(string) string
}

/*
Run 使用 workerCount 线程，以 entry 作为入口，进行遍历，使用  shouldContinueOnUrl 判断这个 url 是否需要深入
*/
func (w *Worker) Run(entryURL string,
	workerCount int,
	shouldContinueOnURL func(string) bool,
	configHTMLWorker func(*HtmlWorker.Worker),
	urlConvert func(string) string,
	finish func()) {

	w.visitedURLList = []string{}
	w.listLock = make(chan int, 1)
	w.listLock <- 1

	w.shouldContinueOnURL = shouldContinueOnURL
	w.configHTMLWorker = configHTMLWorker
	w.urlConvert = urlConvert

	w.taskQueue = AsynWorker.New()
	w.taskQueue.MaxRoutineCount = workerCount
	w.taskQueue.StopedAction = finish

	w.taskQueue.AddHandlerTask(func() {
		w.runPage(entryURL, w.shouldContinueOnURL, w.configHTMLWorker, w.urlConvert)
	})
}

func (w *Worker) runPage(url string,
	shouldContinueOnURL func(string) bool,
	configHTMLWorker func(*HtmlWorker.Worker),
	urlConvert func(string) string) {

	action := HtmlWorker.NewAction("a", func(sel *goquery.Selection) {
		sel.Each(func(index int, s *goquery.Selection) {
			href, isexist := s.Attr("href")
			if isexist {
				if shouldContinueOnURL(href) {
					if w.addURLUnVisitedIfNoExist(href) == true {
						href = urlConvert(href)
						w.taskQueue.AddHandlerTask(func() {
							w.runPage(href, w.shouldContinueOnURL, w.configHTMLWorker, w.urlConvert)
						})
					}
				}
			}
		})
	})

	worker := HtmlWorker.New(url, []HtmlWorker.WorkerAction{action})
	configHTMLWorker(&worker)
	worker.OnFail = func(err error) {
		w.runningCount--
		if w.runningCount <= 0 {
			w.OnFinish()
		}
	}
	worker.OnFinish = func() {
		w.runningCount--
		if w.runningCount <= 0 {
			w.OnFinish()
		}
	}

	w.runningCount++

	fmt.Printf("fetch url %s\n", url)
	worker.Run()
}

func (w *Worker) addURLUnVisitedIfNoExist(url string) bool {
	<-w.listLock
	for i := 0; i < len(w.visitedURLList); i++ {
		if w.visitedURLList[i] == url {
			w.listLock <- 1
			return false
		}
	}
	w.visitedURLList = append(w.visitedURLList, url)
	w.listLock <- 1
	return true
}

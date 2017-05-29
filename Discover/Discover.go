package Discover

import (
	"fmt"

	"time"

	"../AsynWorker"
	"../HtmlWorker"
	"github.com/PuerkitoBio/goquery"
)

/*
Worker is a queue, which can be added in some task to be excuted
*/
type Worker struct {
	visitedURLList []string
	listLock       chan int
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

	taskQueue := AsynWorker.New()
	taskQueue.MaxRoutineCount = workerCount
	taskQueue.StopedActopn = finish

	taskQueue.AddHandlerTask(func() {
		w.runPage(taskQueue,
			entryURL,
			shouldContinueOnURL,
			configHTMLWorker,
			urlConvert)
	})
}

func (w *Worker) runPage(asynWorker AsynWorker.AsynWorker,
	url string,
	shouldContinueOnURL func(string) bool,
	configHTMLWorker func(*HtmlWorker.Worker),
	urlConvert func(string) string) {

	action := HtmlWorker.NewAction("a", func(sel *goquery.Selection) {
		sel.Each(func(index int, s *goquery.Selection) {
			href, isexist := s.Attr("href")
			if isexist {
				if shouldContinueOnURL(href) {
					if w.addURLUnVisitedIfNoExist(href) == false {
						fmt.Printf("访问过的url: %s\n", href)
						return
					}
					href = urlConvert(href)
					time.Sleep(time.Second) //停止一秒，减少爬虫对服务器压力
					asynWorker.AddHandlerTask(func() {
						w.runPage(asynWorker,
							href,
							shouldContinueOnURL,
							configHTMLWorker,
							urlConvert)
					})
				}
				fmt.Printf("放弃url: %s\n", href)
			}
		})
	})
	worker := HtmlWorker.New(url, []HtmlWorker.WorkerAction{action})
	configHTMLWorker(&worker)
	worker.OnFail = func(err error) {
		fmt.Println(err)
	}
	worker.OnFinish = func() {
	}

	fmt.Printf("start fetch %s\n", url)
	worker.Run()
}

func (w *Worker) addURLUnVisitedIfNoExist(url string) bool {
	<-w.listLock
	fmt.Println("lock")
	for i := 0; i < len(w.visitedURLList); i++ {
		if w.visitedURLList[i] == url {
			w.listLock <- 1
			fmt.Println("unlock")
			return false
		}
	}
	w.visitedURLList = append(w.visitedURLList, url)
	w.listLock <- 1
	fmt.Println("unlock")
	return true
}

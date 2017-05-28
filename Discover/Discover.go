package Discover

import (
	"fmt"

	"../AsynWorker"
	"../HtmlWorker"
	"github.com/PuerkitoBio/goquery"
)

/*
Worker is a queue, which can be added in some task to be excuted
*/
type Worker struct {
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
					href = urlConvert(href)
					asynWorker.AddHandlerTask(func() {
						w.runPage(asynWorker,
							href,
							shouldContinueOnURL,
							configHTMLWorker,
							urlConvert)
					})
				}
			}
		})
	})
	worker := HtmlWorker.New(url, []HtmlWorker.WorkerAction{action})
	configHTMLWorker(&worker)
	worker.OnFail = func(err error) {
		fmt.Println(err)
		// fmt.Printf("faech fail %s\n", url)
	}
	worker.OnFinish = func() {
		// fmt.Printf("end fetch %s\n", url)
	}

	fmt.Printf("start fetch %s\n", url)
	worker.Run()
}

package YZY

import (
	"fmt"
	"strings"

	"../Discover"
	"../HtmlWorker"
)

/*
Run 开启一个爬虫
*/
func Run() {
	ch := make(chan int, 1)
	d := Discover.Worker{}
	d.Run("http://www.youzy.cn/",
		100,
		func(url string) bool {
			return isInsiteURL(url) == true && isPicURL(url) == false && isJSCSSURL(url) == false
		},
		configHTMLWorker,
		func(url string, worker *HtmlWorker.Worker) {
			fmt.Println(url)
		},
		fullURL,
		func() {
			ch <- 1
		})
	<-ch
}

func isInsiteURL(URL string) bool {
	if strings.HasPrefix(URL, "/") ||
		strings.HasPrefix(URL, "../") ||
		strings.HasPrefix(URL, "http://www.youzy.cn") ||
		strings.HasPrefix(URL, "www.youzy.cn") {
		return true
	}
	return false
}

func fullURL(url string) string {
	if strings.HasPrefix(url, "www.") {
		return url
	}
	if strings.HasPrefix(url, "http://") {
		return url
	}
	if strings.HasPrefix(url, "/") {
		return "http://www.youzy.cn" + url
	}
	if strings.HasPrefix(url, "./") {
		url = strings.TrimLeft(url, ".")
		return "http://www.youzy.cn" + url
	}
	fmt.Printf("无法处理的url: %s\n", url)
	return url
}

func isPicURL(URL string) bool {
	if strings.HasSuffix(URL, ".jpg") ||
		strings.HasSuffix(URL, ".png") {
		return true
	}
	return false
}

func isJSCSSURL(URL string) bool {
	if strings.HasSuffix(URL, ".js") ||
		strings.HasSuffix(URL, ".css") {
		return true
	}
	return false
}

func configHTMLWorker(worker *HtmlWorker.Worker) {
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		fmt.Println("================================")
		fmt.Printf("content length %d\n", len(buffer))
		return buffer, nil
	}
}

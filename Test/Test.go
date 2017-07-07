package Test

import (
	"fmt"
	"os"
	"strings"

	"../Encoding"
	"../HtmlWorker"
	"github.com/PuerkitoBio/goquery"
)

/*
	Run start task
*/
func Run() {
	action := HtmlWorker.NewAction("#chapterList > li > a", func(sel *goquery.Selection) {
		urls := []string{}
		sel.Each(func(index int, s *goquery.Selection) {
			url, isExist := s.Attr("href")
			if isExist {
				url = fullURL(url)
				urls = append(urls, url)
			}
		})
		downloadChapters(urls)
	})

	worker := HtmlWorker.New("http://www.uukanshu.net/b/43066/", []HtmlWorker.WorkerAction{action})
	worker.CookieStrig = "lastread=11356%3D0%3D%7C%7C17203%3D0%3D%7C%7C17151%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; ASP.NET_SessionId=fm1nai0bstdsevx2zoxva3vh; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.779825662.1496043539; fcip=111"
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		return Encoding.GbkToUtf8(buffer)
	}
	worker.Run()
}

func downloadChapters(urls []string) {
	file, err := os.Open("/Users/karsa/Downloads/ttt.txt")
	if nil != err {
		fmt.Println(err)
		return
	}
	defer file.Close()

	for i := 0; i < len(urls); i++ {
		contentAction := HtmlWorker.NewAction("#contentbox", func(sel *goquery.Selection) {
			content := sel.Text()
			file.Write([]byte(content))
		})
		url := urls[len(urls)-i-1]
		fmt.Printf("start chapter %s\n", url)
		worker := HtmlWorker.New(url, []HtmlWorker.WorkerAction{contentAction})
		worker.CookieStrig = "lastread=11356%3D0%3D%7C%7C17203%3D0%3D%7C%7C17151%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; ASP.NET_SessionId=fm1nai0bstdsevx2zoxva3vh; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.779825662.1496043539; fcip=111"
		worker.Encoder = func(buffer []byte) ([]byte, error) {
			return Encoding.GbkToUtf8(buffer)
		}
		worker.Run()
	}
}

func fullURL(url string) string {
	if strings.HasPrefix(url, "www.") {
		return url
	}
	if strings.HasPrefix(url, "http://") {
		return url
	}
	if strings.HasPrefix(url, "/") {
		return "http://www.uukanshu.net" + url
	}
	if strings.HasPrefix(url, "./") {
		url = strings.TrimLeft(url, ".")
		return "http://www.uukanshu.net" + url
	}
	fmt.Printf("无法处理的url: %s\n", url)
	return url
}

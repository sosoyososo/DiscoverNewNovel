package main

/*
	Author : Karsa
	Data : 2017-5-6
*/

/*
主体逻辑:
	2. 获取目录所在URL，根据规则分析出目录列表的标题和URL
	3. 读取各章节内容
	4. 分析网页内容提取征文信息
TODO:
	1. 保存整体信息，图片，标题，作者等
	2. 支持增量更新
		a. 将分析出来的章节列表保存到数据库中
		b. 标记已经下载好的章节
		c. 下次下载读取数据库，然后获取内容没有获取成功的章节，只获取没有获取成功的内容
*/
import (
	"fmt"

	"./Encoding"
	"./HtmlWorker"
	"./MongoDB"
	"./Uukanshu"
	"github.com/PuerkitoBio/goquery"
)

func main() {
	loadData()
	// test()
}

func loadData() {
	ch := make(chan int, 1)
	Uukanshu.Run("http://www.uukanshu.net/b/11356/", 3, func() {
		ch <- 1
	})
	<-ch
	fmt.Println("结束")
}

func test() {
	contentAction := HtmlWorker.NewAction("#chapterList > li", func(s *goquery.Selection) {
		s.Each(func(index int, sel *goquery.Selection) {
			content := s.Text()
			fmt.Println(content)
		})
	})

	url := "http://www.uukanshu.net/b/11356/"
	worker := HtmlWorker.New(url, []HtmlWorker.WorkerAction{contentAction})
	worker.CookieStrig = "ASP.NET_SessionId=33o4lgiftcbae54smwa1cbzk; lastread=11356%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.475465301.1495266639; fcip=111"
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		return Encoding.GbkToUtf8(buffer)
	}
	worker.Run()
}

func testMongo() {
	MongoDB.TestMongo()
}

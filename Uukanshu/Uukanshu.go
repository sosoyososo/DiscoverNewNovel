package Uukanshu

import (
	"fmt"
	"strings"

	"../AsynWorker"
	"../Discover"
	"../Encoding"
	"../HtmlWorker"
	"github.com/PuerkitoBio/goquery"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

/*
ChapterInfo 保存章节信息
*/
type ChapterInfo struct {
	CateURL string
	Title   string
	URL     string
	Index   int
}

/*
NovelInfo 保存小说信息
*/
type NovelInfo struct {
	URL      string
	Title    string
	Author   string
	Summary  string
	CoverImg string
	HasInfo  bool
}

/*
Action 获取某个章节的内容
*/
func (c ChapterInfo) Action() {
	contentAction := HtmlWorker.NewAction("div.contentbox", func(s *goquery.Selection) {
		fmt.Println(c.Title)
	})
	worker := HtmlWorker.New(c.URL, []HtmlWorker.WorkerAction{contentAction})
	worker.CookieStrig = "lastread=11356%3D0%3D%7C%7C17203%3D0%3D%7C%7C17151%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; ASP.NET_SessionId=fm1nai0bstdsevx2zoxva3vh; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.779825662.1496043539; fcip=111"
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		return Encoding.GbkToUtf8(buffer)
	}
	worker.Run()
}

var (
	dbSession       *mgo.Session
	novelDb         *mgo.Database
	dbWorker        *AsynWorker.SynWorker
	novelInfoWorker *AsynWorker.AsynWorker
)

/*
RunSpider 以某个页面作为入口启动一个蜘蛛，爬取所有的目录页面
注意:
	站内搜素所有非小说详情的页面
	每个页面1个小时内最多遍历1次
	发现新的小说目录页面，应该发出通知
*/

/*
DiscoverNewChapters 遍历所有小说，获取目录页，遍历章节，发现新的章节
*/
// TODO: 获取每个小说的章节列表
// TODO: 发现新的章节
// TODO: 写入数据库
func DiscoverNewChapters(finish func()) {
	connectToDbIfNeed()
	createDBWorkerInfoNeeded()

	novelCollection := novelDb.C("novels")
	iter := novelCollection.Find(bson.M{}).Iter()

	asynWorker := AsynWorker.New()
	result := NovelInfo{}

	count := 0
	for iter.Next(&result) {
		retURL := result.URL
		if len(retURL) > 0 {
			count++
			asynWorker.AddHandlerTask(func() {
				findChaptersForNovel(retURL, func() {
					count--
					if count <= 0 {
						finish()
					}
				})
			})
		}
	}
}

func findChaptersForNovel(cateURL string, finish func()) {
	connectToDbIfNeed()
	createDBWorkerInfoNeeded()

	novelCollection := novelDb.C("chapters")
	query := novelCollection.Find(bson.M{"cateurl": cateURL})

	chaptersAction := HtmlWorker.NewAction("#chapterList > li > a", func(sel *goquery.Selection) {
		length := len(sel.Nodes)
		sel.Each(func(index int, s *goquery.Selection) {
			url, isExist := s.Attr("href")
			if isExist {
				url = fullURL(url)

				chapterInfo := ChapterInfo{}
				iter := query.Iter()
				for iter.Next(&chapterInfo) {
					if chapterInfo.URL == url { //数据库已经存在相同的章节
						return
					}
				}

				chapterIndex := length - index

				chapterInfo.URL = url
				chapterInfo.CateURL = cateURL
				chapterInfo.Index = chapterIndex
				chapterInfo.Title = s.Text()
				err := novelCollection.Insert(chapterInfo)
				if err != nil {
					fmt.Printf(" %s 插入 第%d章节失败 %s　\n", cateURL, chapterIndex, err.Error())
				}
			}
		})
	})

	worker := HtmlWorker.New(cateURL, []HtmlWorker.WorkerAction{chaptersAction})
	configHTMLWorker(&worker)
	worker.OnFail = func(err error) {
		fmt.Printf("更新 %s 失败　%s\n", cateURL, err.Error())
		finish()
	}
	worker.OnFinish = func() {
		fmt.Printf("完成 %s 的更新\n", cateURL)
		finish()
	}
	worker.Run()
}

/*
CollecteNovelInfo 遍历数据库，获取每个小说的信息
*/
func CollecteNovelInfo(finish func()) {
	connectToDbIfNeed()
	createDBWorkerInfoNeeded()

	novelCollection := novelDb.C("novels")
	iter := novelCollection.Find(bson.M{"hasinfo": false}).Iter()

	asynWorker := AsynWorker.New()
	result := NovelInfo{}

	runingCount := 0
	for iter.Next(&result) {
		runingCount++
		cateURL := result.URL
		asynWorker.AddHandlerTask(func() {
			runNovelInfoFetch(cateURL, func() {
				runingCount--
				if runingCount <= 0 && finish != nil {
					finish()
				}
			})
		})
	}
	if err := iter.Close(); err != nil {
		fmt.Println("关闭数据库查询遍历器失败")
	}
	if runingCount <= 0 && finish != nil {
		finish()
	}
}

/*
RunSpider 启动发现小说的爬虫
*/
func RunSpider(finished func()) {
	connectToDbIfNeed()
	createDBWorkerInfoNeeded()

	d := Discover.Worker{}
	d.Run("http://www.uukanshu.net/sitemap/novellist-1.html",
		20,
		func(url string) bool {
			return isInsiteURL(url) == true && isPicURL(url) == false
		},
		configHTMLWorker,
		func(url string) string {
			if isCatelogURL(url) { //如果是目录URL，走找到小说的路径
				foundNovelURL(url)
			}
			return fullURL(url)
		},
		finished)
}

/*
foundNovelURL 在 RunSpider 过程中发现新的小说URL，需要插入到数据库中
*/
func foundNovelURL(catelogURL string) {
	dbWorker.AddAction(func() {
		novels := novelDb.C("novels")
		cateURL := fullURL(catelogURL)
		count, err := novels.Find(bson.M{"url": cateURL}).Count()
		if nil != err || count == 0 { //找到新的小说后，获取小说信息，将之更新到数据库
			fmt.Printf("发现新小说:%s\n", catelogURL)

			novel := NovelInfo{}
			novel.URL = cateURL
			novelCollection := novelDb.C("novels")

			err := novelCollection.Insert(&novel)
			if err != nil {
				fmt.Println("插入小说失败")
			}
		}
	})
}

/*
RunCateFetch 使用一个 Uukanshu 的目录页面 url，读取小说信息，读取目录列表
*/
func runNovelInfoFetch(cateURL string, finished func()) {
	cateURL = fullURL(cateURL)
	novelCollection := novelDb.C("novels")

	novelInfo := NovelInfo{}
	novelInfo.URL = cateURL

	titleAction := HtmlWorker.NewAction("dd > h1 > a", func(s *goquery.Selection) {
		novelInfo.Title = s.Text()
	})
	coverAction := HtmlWorker.NewAction(".jieshao > dt > a > img", func(s *goquery.Selection) {
		url, isExist := s.Attr("src")
		if isExist {
			novelInfo.CoverImg = url
		}
	})
	summaryAction := HtmlWorker.NewAction("dd > h3", func(s *goquery.Selection) {
		novelInfo.Summary = s.Text()
	})
	authorAction := HtmlWorker.NewAction("dd > h2 > a", func(s *goquery.Selection) {
		novelInfo.Author = s.Text()
	})

	worker := HtmlWorker.New(cateURL, []HtmlWorker.WorkerAction{titleAction, coverAction, authorAction, summaryAction})
	configHTMLWorker(&worker)
	worker.OnFail = func(err error) {
		fmt.Printf("fail on : %s with error : %s\n", cateURL, err.Error())
		finished()
	}
	worker.OnFinish = func() {
		fmt.Printf("Add Action : %s\n", cateURL)
		dbWorker.AddAction(func() {
			err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"title": novelInfo.Title, "author": novelInfo.Author, "summary": novelInfo.Summary, "coverimg": novelInfo.CoverImg, "hasinfo": true}})
			if nil != err {
				fmt.Printf("update info fail on : %s\n", cateURL)
			} else {
				fmt.Printf("update info succeed on : %s\n", cateURL)
			}
			finished()
		})
	}
	worker.Run()
}

func connectToDbIfNeed() {
	if dbSession == nil {
		dbSession, err := mgo.Dial("127.0.0.1:27017")
		if err != nil {
			panic(err)
		}
		novelDb = dbSession.DB("novel")
	}
}

func createDBWorkerInfoNeeded() {
	if dbWorker == nil {
		dbWorker = &AsynWorker.SynWorker{}
	}
}

func isInsiteURL(URL string) bool {
	if strings.HasPrefix(URL, "/") ||
		strings.HasPrefix(URL, "../") ||
		strings.HasPrefix(URL, "http://www.uukanshu.net") ||
		strings.HasPrefix(URL, "www.uukanshu.net") {
		return true
	}
	return false
}

func isPicURL(URL string) bool {
	if strings.HasSuffix(URL, ".jpg") ||
		strings.HasSuffix(URL, ".png") {
		return true
	}
	return false
}

func isCatelogURL(URL string) bool {
	if strings.HasPrefix(URL, "/b") {
		if strings.HasSuffix(URL, ".html") {
			return false
		}
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
		return "http://www.uukanshu.net" + url
	}
	if strings.HasPrefix(url, "./") {
		url = strings.TrimLeft(url, ".")
		return "http://www.uukanshu.net" + url
	}
	fmt.Printf("无法处理的url: %s\n", url)
	return url
}

func configHTMLWorker(worker *HtmlWorker.Worker) {
	worker.CookieStrig = "lastread=11356%3D0%3D%7C%7C17203%3D0%3D%7C%7C17151%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; ASP.NET_SessionId=fm1nai0bstdsevx2zoxva3vh; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.779825662.1496043539; fcip=111"
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		return Encoding.GbkToUtf8(buffer)
	}
}

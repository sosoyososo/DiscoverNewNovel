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
NovelInfo 保存小说信息
*/
type NovelInfo struct {
	URL      string
	Title    string
	Author   string
	Summary  string
	CoverImg string
	HasInfo  bool
	Tags     []string
}

/*
ChapterInfo 保存章节信息
*/
type ChapterInfo struct {
	CateURL string
	Title   string
	URL     string
	Index   int
}

var (
	dbSession       *mgo.Session
	novelDb         *mgo.Database
	dbWorker        *AsynWorker.SynWorker
	novelInfoWorker *AsynWorker.AsynWorker
)

// 发现新的小说
// 开始后每24小时执行一次
func Run() {
	fmt.Print("=============== 开始寻找小说 ========================")
	ch := make(chan int, 1)
	RunSpider(func() {
		ch <- 1
	})
	<-ch

	collectionNovelsInfo()
}

// 搜集每本小说的信息
func collectionNovelsInfo() {
	fmt.Print("=============== 开始完善小说信息 ========================")
	ch := make(chan int, 1)
	CollecteNovelInfo(func() {
		ch <- 1
	})
	<-ch

	runChapterDiscovery()
}

// 发现新的章节
// 开始后每隔小时执行一次
func runChapterDiscovery() {
	fmt.Print("=============== 开始完善章节信息 ========================")

	ch := make(chan int, 1)
	DiscoverNewChapters(func() {
		ch <- 1
	})
	<-ch
	fmt.Println("finished")
}

/*
==================================================================================
小说发现逻辑
==================================================================================
*/

/*
RunSpider 以某个页面作为入口启动一个蜘蛛，爬取所有的目录页面
注意:
	站内搜素所有非小说详情的页面
	每个页面1个小时内最多遍历1次
	发现新的小说目录页面，应该发出通知
*/
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
				foundNovel(url)
			}
			return fullURL(url)
		},
		finished)
}

/*
foundNovel 在 RunSpider 过程中发现新
*/
func foundNovel(catelogURL string) {
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
==================================================================================
小说章节发现逻辑
==================================================================================
*/

/*
DiscoverNewChapters 遍历所有小说，获取目录页，遍历章节，发现新的章节
*/
func DiscoverNewChapters(finish func()) {
	connectToDbIfNeed()
	createDBWorkerInfoNeeded()

	novelCollection := novelDb.C("novels")
	iter := novelCollection.Find(bson.M{}).Iter()

	asynWorker := AsynWorker.New()
	asynWorker.MaxRoutineCount = 10
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
	count, err := query.Count()

	chaptersAction := HtmlWorker.NewAction("#chapterList > li > a", func(sel *goquery.Selection) {
		length := len(sel.Nodes)
		if err == nil {
			if length == count { //已经存储的内容和现有内容数量一致，不需要更新
				return
			}
		}

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
				novelCollection.Insert(chapterInfo)
			}
		})
	})

	worker := HtmlWorker.New(cateURL, []HtmlWorker.WorkerAction{chaptersAction})
	configHTMLWorker(&worker)
	worker.OnFail = func(err error) {
		finish()
	}
	worker.OnFinish = func() {
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
	iter.Close()
	if runingCount <= 0 && finish != nil {
		finish()
	}
}

/*
RunCateFetch 使用一个 Uukanshu 的目录页面 url，读取小说信息，读取目录列表
*/
func runNovelInfoFetch(cateURL string, finished func()) {
	cateURL = fullURL(cateURL)
	novelCollection := novelDb.C("novels")

	novelInfo := NovelInfo{}
	novelInfo.URL = cateURL

	statusAction := HtmlWorker.NewAction(".status-text", func(s *goquery.Selection) {
		status := s.Text()
		if len(status) > 0 {
			for i := 0; i < len(novelInfo.Tags); i++ {
				if novelInfo.Tags[i] == status {
					return
				}
			}

			novelInfo.Tags = append(novelInfo.Tags, status)
		}
	})
	titleAction := HtmlWorker.NewAction("dd > h1 > a", func(s *goquery.Selection) {
		title := s.Text()
		novelInfo.Title = strings.TrimLeft(title, "最新章节")
	})
	coverAction := HtmlWorker.NewAction(".jieshao > dt > a > img", func(s *goquery.Selection) {
		url, isExist := s.Attr("src")
		if isExist {
			novelInfo.CoverImg = url
		}
	})
	summaryAction := HtmlWorker.NewAction("dd > h3", func(s *goquery.Selection) {
		summary := s.Text()
		summary = strings.TrimSpace(summary)
		summary = strings.Trim(summary, "\n")
		summary = strings.Trim(summary, "－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－－")
		summary = strings.ToLower(summary)
		summary = strings.Trim(summary, "http://www.uukanshu.net")
		novelInfo.Summary = summary
	})
	authorAction := HtmlWorker.NewAction("dd > h2 > a", func(s *goquery.Selection) {
		novelInfo.Author = s.Text()
	})

	worker := HtmlWorker.New(cateURL, []HtmlWorker.WorkerAction{titleAction, coverAction, authorAction, summaryAction, statusAction})
	configHTMLWorker(&worker)
	worker.OnFail = func(err error) {
		finished()
	}
	worker.OnFinish = func() {
		dbWorker.AddAction(func() {
			novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"title": novelInfo.Title, "author": novelInfo.Author, "summary": novelInfo.Summary, "coverimg": novelInfo.CoverImg, "hasinfo": true, "tags": novelInfo.Tags}})
			finished()
		})
	}
	worker.Run()
}

/*
==================================================================================
其他支持函数
==================================================================================
*/

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

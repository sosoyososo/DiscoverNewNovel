package Uukanshu

import (
	"fmt"
	"strings"
	"time"

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
Action 获取章节内容
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

/*
NovelInfo 保存小说信息
*/
type NovelInfo struct {
	URL      string
	Title    string
	Author   string
	Summary  string
	CoverImg string
}

type spiderRecorder struct {
	URL  string
	Time string //yyyy-MM-dd-hh-mm-ss
}

func (s *spiderRecorder) getTime() *time.Time {
	if len(s.Time) > 0 {
		const longForm = "Jan 2, 2006 at 3:04pm (MST)"
		t, err := time.Parse(longForm, s.Time)
		if nil == err {
			return &t
		}
	}
	return nil
}

func (s *spiderRecorder) setTime(time.Time) {
	const longForm = "Jan 2, 2006 at 3:04pm (MST)"
	s.Time = time.Now().Format(longForm)
}

var (
	dbSession       *mgo.Session
	novelDb         *mgo.Database
	dbWorker        *AsynWorker.SynWorker
	novelInfoWorker *AsynWorker.AsynWorker
)

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

/*
RunCateFetch 使用一个 Uukanshu 的目录页面 url，读取小说信息，读取目录列表
*/
func runNovelInfoFetch(cateURL string, routineCount int, finished func()) {
	connectToDbIfNeed()
	db := novelDb

	cateURL = fullURL(cateURL)
	novelCollection := db.C("novels")

	titleAction := HtmlWorker.NewAction("dd > h1 > a", func(s *goquery.Selection) {
		dbWorker.AddAction(func() {
			err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"title": s.Text()}})
			if err != nil {
				fmt.Println("标题更新失败")
			}
			fmt.Println("更新标题")
		})
	})
	coverAction := HtmlWorker.NewAction(".jieshao > dt > a > img", func(s *goquery.Selection) {
		url, isExist := s.Attr("src")
		if isExist {
			dbWorker.AddAction(func() {
				err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"coverimg": url}})
				if err != nil {
					fmt.Println("封面更新失败")
				}
				fmt.Println("更新封面")
			})
		}
	})
	summaryAction := HtmlWorker.NewAction("dd > h3", func(s *goquery.Selection) {
		dbWorker.AddAction(func() {
			err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"summary": s.Text()}})
			if err != nil {
				fmt.Println("摘要更新失败")
			}
			fmt.Println("更新摘要")
		})
	})
	authorAction := HtmlWorker.NewAction("dd > h2 > a", func(s *goquery.Selection) {
		dbWorker.AddAction(func() {
			err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"author": s.Text()}})
			if err != nil {
				fmt.Println("作者更新失败")
			}
			fmt.Println("更新作者")
		})
	})

	fmt.Printf("发现新小说%s\n", cateURL)
	worker := HtmlWorker.New(cateURL, []HtmlWorker.WorkerAction{titleAction, coverAction, authorAction, summaryAction})
	worker.CookieStrig = "lastread=11356%3D0%3D%7C%7C17203%3D0%3D%7C%7C17151%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; ASP.NET_SessionId=fm1nai0bstdsevx2zoxva3vh; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.779825662.1496043539; fcip=111"
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		return Encoding.GbkToUtf8(buffer)
	}
	if novelInfoWorker == nil {
		asynWorker := AsynWorker.New()
		novelInfoWorker = &asynWorker
	}
	novelInfoWorker.AddHandlerTask(func() {
		worker.Run()
	})
}

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
// TODO: 遍历数据库所有小说
// TODO: 获取每个小说的章节列表
// TODO: 发现新的章节
// TODO: 写入数据库
func DiscoverNewChapters(finish func()) {
}

/*
RunSpider 启动发现小说的爬虫
*/
func RunSpider(finished func()) {
	connectToDbIfNeed()
	createDBWorkerInfoNeeded()

	d := Discover.Worker{}
	d.Run("http://www.uukanshu.net/",
		4,
		func(url string) bool {
			return isInsiteURL(url) == true && isPicURL(url) == false
		},
		func(worker *HtmlWorker.Worker) {
			worker.CookieStrig = "lastread=11356%3D0%3D%7C%7C17203%3D0%3D%7C%7C17151%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; ASP.NET_SessionId=fm1nai0bstdsevx2zoxva3vh; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.779825662.1496043539; fcip=111"
			worker.Encoder = func(buffer []byte) ([]byte, error) {
				return Encoding.GbkToUtf8(buffer)
			}
		},
		func(url string) string {
			if isCatelogURL(url) { //如果是目录URL，走找到小说的路径
				findNovelURL(url)
			}
			return fullURL(url)
		},
		finished)
}

func isInsiteURL(URL string) bool {
	if strings.HasPrefix(URL, "/") || strings.HasPrefix(URL, "../") || strings.HasPrefix(URL, "http://www.uukanshu.net") || strings.HasPrefix(URL, "www.uukanshu.net") {
		return true
	}
	return false
}

func isPicURL(URL string) bool {
	if strings.HasSuffix(URL, ".jpg") || strings.HasSuffix(URL, ".png") {
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
	fmt.Println("无法处理的url: " + url)
	return url
}

func findNovelURL(catelogURL string) {
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

package Uukanshu

import (
	"fmt"
	"strings"
	"time"

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
}

/*
Action 获取章节内容
*/
func (c ChapterInfo) Action() {
	contentAction := HtmlWorker.NewAction("div.contentbox", func(s *goquery.Selection) {
		fmt.Println(c.Title)
	})
	worker := HtmlWorker.New(c.URL, []HtmlWorker.WorkerAction{contentAction})
	worker.CookieStrig = "ASP.NET_SessionId=azxql35ktrk12lqlleegfrus; lastread=55516%3D0%3D; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.1926814091.1494322381; fcip=111"
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		return Encoding.GbkToUtf8(buffer)
	}
	worker.Run()
}

/*
RunCateFetch 使用一个 Uukanshu 的目录页面 url，读取小说信息，读取目录列表
*/
func RunCateFetch(cateURL string, routineCount int, finished func()) {
	// 打开数据库连接
	session, err := mgo.Dial("127.0.0.1:27017")
	if err != nil {
		panic(err)
	}
	db := session.DB("novel")

	novel := NovelInfo{}
	novel.URL = cateURL
	novelCollection := db.C("novels")
	err = novelCollection.Insert(&novel)
	if err != nil {
		fmt.Println("DB 操作失败")
		return
	}

	titleAction := HtmlWorker.NewAction("dd > h1 > a", func(s *goquery.Selection) {
		err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"title": s.Text()}})
		if err != nil {
			fmt.Println("标题更新失败")
		}
	})
	coverAction := HtmlWorker.NewAction(".jieshao > dt > a > img", func(s *goquery.Selection) {
		url, isExist := s.Attr("src")
		if isExist {
			err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"coverimg": url}})
			if err != nil {
				fmt.Println("封面更新失败")
			}
		}
	})
	summaryAction := HtmlWorker.NewAction("dd > h3", func(s *goquery.Selection) {
		err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"summary": s.Text()}})
		if err != nil {
			fmt.Println("摘要更新失败")
		}
	})
	authorAction := HtmlWorker.NewAction("dd > h2 > a", func(s *goquery.Selection) {
		err := novelCollection.Update(bson.M{"url": cateURL}, bson.M{"$set": bson.M{"author": s.Text()}})
		if err != nil {
			fmt.Println("作者更新失败")
		}
	})
	chaptersAction := HtmlWorker.NewAction("#chapterList > li > a", func(s *goquery.Selection) {
		c := db.C("chapters")
		s.Each(func(index int, aTag *goquery.Selection) {
			// aTag := s.Find("a")
			url, isExist := aTag.Attr("href")
			title := aTag.Text()
			fmt.Printf("第%d个插入\n", index)
			if isExist {
				chapter := ChapterInfo{}
				chapter.CateURL = cateURL
				chapter.Index = index
				chapter.URL = "http://www.uukanshu.net" + url
				chapter.Title = title

				err := c.Insert(&chapter)
				if err != nil {
					fmt.Printf("DB 操作失败 : %s %s\n ", title, err.Error())
				} else {
					fmt.Println(title)
				}
			}
		})

		if finished != nil {
			finished()
		}
	})

	worker := HtmlWorker.New(cateURL, []HtmlWorker.WorkerAction{titleAction, coverAction, authorAction, summaryAction, chaptersAction})
	worker.CookieStrig = "ASP.NET_SessionId=azxql35ktrk12lqlleegfrus; lastread=55516%3D0%3D; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.1926814091.1494322381; fcip=111"
	worker.Encoder = func(buffer []byte) ([]byte, error) {
		return Encoding.GbkToUtf8(buffer)
	}
	go worker.Run()
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

/*
RunSpider 以某个页面作为入口启动一个蜘蛛，爬取所有的目录页面
注意:
	站内搜素所有非小说详情的页面
	每个页面1个小时内最多遍历1次
	发现新的小说目录页面，应该发出通知
*/
func RunSpider(finished func()) {
	session, err := mgo.Dial("127.0.0.1:27017")
	if err != nil {
		panic(err)
	}
	db := session.DB("novel")
	uuCollection := db.C("uukanshu_urls")

	d := Discover.Worker{}
	d.Run("http://www.uukanshu.net/",
		4,
		func(url string) bool {
			fmt.Println(url)
			if strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") {
				query := uuCollection.Find(bson.M{"url": url})
				count, err := query.Count()
				if err == nil {
					if count > 0 {
						record := spiderRecorder{}
						query.One(&record)

						t := record.getTime()
						if t != nil {
							duration := time.Since(*t)
							if duration < 60*60 {
								return false
							}
						}
					}
				}
				return true
			}
			return false
		},
		func(worker *HtmlWorker.Worker) {
			worker.CookieStrig = "ASP.NET_SessionId=33o4lgiftcbae54smwa1cbzk; lastread=11356%3D0%3D%7C%7C17203%3D0%3D%7C%7C17151%3D0%3D%7C%7C482%3D0%3D%7C%7C55516%3D10981%3D%u7B2C8%u7AE0%20%u5C38%u53D8; fcip=111; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.458374049.1495553756; _gat=1"
			worker.Encoder = func(buffer []byte) ([]byte, error) {
				return Encoding.GbkToUtf8(buffer)
			}
		},
		func(url string) string {
			record := spiderRecorder{}
			record.URL = url
			record.setTime(time.Now())

			query := uuCollection.Find(bson.M{"url": url})
			count, _ := query.Count()
			if count > 0 {
				uuCollection.Update(bson.M{"url": url}, bson.M{"$set": bson.M{"time": record.Time}})
			} else {
				uuCollection.Insert(&record)
			}

			if strings.HasPrefix(url, "/") {
				return "http://www.uukanshu.net" + url
			}

			return url
		},
		finished)

}

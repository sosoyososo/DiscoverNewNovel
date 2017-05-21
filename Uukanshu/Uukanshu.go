package Uukanshu

import (
	"fmt"
	"strings"

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
ChapterInfo 保存小说信息
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
Run 使用一个 Uukanshu 的目录页面 url，读取小说信息，读取目录列表
*/
func Run(cateURL string, routineCount int, finished func()) {
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

func absoluteURL(url string) string {
	baseURL := ""
	if strings.HasPrefix(url, "/") {
		return baseURL + url
	} else if strings.HasPrefix(url, "./") {
		return baseURL + strings.TrimPrefix(url, ".")
	}
	return url
}

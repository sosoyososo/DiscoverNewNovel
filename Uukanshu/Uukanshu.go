package Uukanshu

import (
	"fmt"
	"strings"

	"../Discover"
	"../Encoding"
	"../HtmlWorker"
	"../MongoDb"
	"github.com/PuerkitoBio/goquery"
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
	novelDbList []NovelInfo //数据库表中小说地址列表
)

/*
Run 是uukanshu总的入口
*/
func Run() {
	initDb()

	fmt.Println("=============== 开始寻找小说 ========================")
	ch := make(chan int, 1)
	RunSpider(func() {
		ch <- 1
	})
	<-ch
}

/*
==================================================================================
小说发现逻辑
==================================================================================
*/

/*
RunSpider 以某个页面作为入口启动一个蜘蛛，爬取所有的目录页面，从目录页面发现所有的小说信息和章节信息
*/
func RunSpider(finished func()) {
	d := Discover.Worker{}
	d.Run("http://www.uukanshu.com/sitemap/novellist-1.html",
		20,
		func(url string) bool {
			return isInsiteURL(url) == true && isPicURL(url) == false
		},
		configHTMLWorker,
		func(cateURL string, worker *HtmlWorker.Worker) {
			if isCatelogURL(cateURL) {
				fmt.Printf("发现小说%s\n", cateURL)
				handleNovelInfo(cateURL, worker)
				handleChapterList(cateURL, worker)
			}
		},
		fullURL,
		finished)
}

func handleChapterList(cateURL string, worker *HtmlWorker.Worker) {
	chaptersAction := HtmlWorker.NewAction("#chapterList > li > a", func(sel *goquery.Selection) {
		chapters := []ChapterInfo{}

		length := len(sel.Nodes)
		sel.Each(func(index int, s *goquery.Selection) {
			url, isExist := s.Attr("href")
			if isExist {
				url = fullURL(url)

				chapterInfo := ChapterInfo{}
				chapterIndex := length - index
				chapterInfo.URL = url
				chapterInfo.CateURL = cateURL
				chapterInfo.Index = chapterIndex
				chapterInfo.Title = s.Text()

				chapters = append(chapters, chapterInfo)
			}
		})

		if len(chapters) > 0 {
			cateURL = fullURL(cateURL)
			chapterCollection := MongoDb.GetUukanshuChapterCollection(cateURL)
			query := chapterCollection.Find(bson.M{"cateurl": cateURL})
			length, err := query.Count()
			if err == nil {
				list := make([]ChapterInfo, length)
				err = query.All(&list)
				if err == nil && len(list) > 0 {
					for i := 0; i < len(chapters); i++ {
						shoudlInsert := true
						for j := 0; j < len(list); j++ {
							if list[i].URL == chapters[j].URL {
								shoudlInsert = false
								break
							}
						}
						if shoudlInsert {
							chapter := chapters[i]
							MongoDb.GetUukanshuChapterCollection(cateURL).Insert(chapter)
						}
					}
				}
			}
		}
	})
	worker.HandleActions([]HtmlWorker.WorkerAction{chaptersAction})
}

func handleNovelInfo(cateURL string, worker *HtmlWorker.Worker) {
	for i := 0; i < len(novelDbList); i++ {
		if novelDbList[i].URL == cateURL {
			fmt.Println("小说重复")
			return
		}
	}

	fmt.Println("插入小说信息")
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
		summary = strings.Trim(summary, "http://www.uukanshu.com")
		novelInfo.Summary = summary
	})
	authorAction := HtmlWorker.NewAction("dd > h2 > a", func(s *goquery.Selection) {
		novelInfo.Author = s.Text()
	})
	worker.HandleActions([]HtmlWorker.WorkerAction{statusAction, titleAction, summaryAction, authorAction, coverAction})
	novelDbList = append(novelDbList, novelInfo)
	MongoDb.GetUukanshuNovelCollection().Insert(&novelInfo)
}

/*
==================================================================================
其他支持函数
==================================================================================
*/

func initDb() {
	query := MongoDb.GetUukanshuNovelCollection().Find(nil)
	count, err := query.Count()
	if nil == err && count > 0 {
		novelDbList = make([]NovelInfo, count)
		query.All(&novelDbList)
	} else {
		novelDbList = []NovelInfo{}
	}
}

func isInsiteURL(URL string) bool {
	if strings.HasPrefix(URL, "/") ||
		strings.HasPrefix(URL, "../") ||
		strings.HasPrefix(URL, "http://www.uukanshu.com") ||
		strings.HasPrefix(URL, "www.uukanshu.com") {
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
	if strings.HasPrefix(URL, "/b") ||
		strings.HasPrefix(URL, "http://www.uukanshu.com/b") ||
		strings.HasPrefix(URL, "www.uukanshu.com/b") {
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
		return "http://www.uukanshu.com" + url
	}
	if strings.HasPrefix(url, "./") {
		url = strings.TrimLeft(url, ".")
		return "http://www.uukanshu.com" + url
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

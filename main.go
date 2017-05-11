package main

/*
	Author : Karsa
	Data : 2017-5-6
*/

/*
主体逻辑:
	2. 获取目录所在URL，根据规则分析出目录列表
	3. 读取各章节内容
	4. 分析网页内容提取征文信息
	5. 自动生成固定格式的配置文件
TODO:
	1. help 支持
	2. 支持目录分页
	3. 支持正文分页
	4. 支持动态渲染的网页(需要使用V8引擎)
	5. 读取配置文件获取需要信息
		1. 获取目录所在网页的URL
		2. 如何获取目录网页的章节URL列表
		3. 如何从网页中获取正文信息
*/
import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"./Data"
	"./Uukanshu"

	"github.com/PuerkitoBio/goquery"
)

var novConfig = Uukanshu.New("http://www.uukanshu.net/b/482/",
	"/Users/karsa/Desktop/tmpNovelDir/")

func readFileContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if nil != err {
		return "", err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if nil != err {
		return "", err
	}

	size := fileInfo.Size()
	buffer := make([]byte, size)
	_, err = file.Read(buffer)
	if nil != err {
		return "", err
	}

	return string(buffer), nil
}

func saveFileContent(content string, fileName string) error {
	fmt.Printf("保存文件 %s\n", fileName)
	savePath := novConfig.Config.SavePath
	filePath := savePath + fileName
	file, err := os.Open(filePath)
	if err != nil {
		file, err = os.Create(filePath)
		if err != nil {
			return err
		}
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

var finishedWorks = make(chan int)

func absoluteURL(url string) string {
	baseURL := novConfig.Config.BaseUrl
	if strings.HasPrefix(url, "/") {
		return baseURL + url
	} else if strings.HasPrefix(url, "./") {
		return baseURL + strings.TrimPrefix(url, ".")
	}
	return url
}

func chapterWorkFailed(title string, url string, err error) {
	fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	fmt.Printf("title: %s, url: %s", title, url)
	fmt.Println(err)
}

func getChapterContent(handler Data.NovelInfoHandler, config Data.NovelConifg, ch chan Data.ChapterConfig) {
	if len(ch) > 0 {
		s := <-ch
		url := handler.ChapterURL(s.S)
		title := handler.ChapterTitle(s.S)
		if len(url) > 0 {
			url = absoluteURL(url)
			buffer, err := getUtf8HtmlBytesFromURL(handler, url)
			if err == nil {
				reader := bytes.NewReader(buffer)
				doc, err := goquery.NewDocumentFromReader(reader)
				if err != nil {
					chapterWorkFailed(title, url, err)
				} else {
					sel := novConfig.Config.ChapterContentSelector
					chapterContent := handler.ChapterContent(doc.Find(sel))
					fileName := fmt.Sprintf("%d-%s.txt", s.Index, title)

					err = saveFileContent(chapterContent, fileName)
					if nil != err {
						fmt.Println(err)
					}
				}
			} else {
				chapterWorkFailed(title, url, err)
			}
		}
		getChapterContent(handler, config, ch)
	} else {
		finishedWorks <- 1
	}
}

func getChaptersContent(handler Data.NovelInfoHandler, config Data.NovelConifg, chapterNodeList *goquery.Selection) {
	nodeCount := len(chapterNodeList.Nodes)
	ch := make(chan Data.ChapterConfig, nodeCount)
	chapterNodeList.Each(func(i int, s *goquery.Selection) {
		ch <- Data.ChapterConfig{nodeCount - i, s}
	})

	l := len(ch)
	count := 0
	for i := 0; i < 3; i++ {
		if l > i {
			count++
			go getChapterContent(handler, config, ch)
		}
	}
	fmt.Printf("%d 个线程进行数据加载", count)
	for i := 0; i < count; i++ {
		<-finishedWorks
		fmt.Printf("============= %d finished ===================", i+1)
	}
}

func getUtf8HtmlBytesFromURL(handler Data.NovelInfoHandler, url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return []byte{}, err
	}

	cookieStr := novConfig.Config.Cookie
	cookieList := strings.Split(cookieStr, ";")
	for i := 0; i < len(cookieList); i++ {
		items := strings.Split(cookieList[i], "=")
		if len(items) >= 2 {
			cookie := http.Cookie{Name: items[0], Value: items[1]}
			req.AddCookie(&cookie)
		}
	}
	tr := &http.Transport{
		DisableCompression: true,
	}

	var client = &http.Client{Transport: tr}
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if strings.HasPrefix(resp.Status, "200") {
		buffer, err := ioutil.ReadAll(resp.Body)
		if len(buffer) <= 0 {
			return nil, err
		}
		buffer, err = handler.ConvertToUtf8(buffer)
		if len(buffer) <= 0 {
			return []byte{}, err
		}
		return buffer, nil
	}
	return []byte{}, errors.New("请求失败")
}

func getNovel(handler Data.NovelInfoHandler, config Data.NovelConifg, doc *goquery.Document) {
	imgSel := config.CoverImageSelector
	doc.Find(imgSel).Each(func(i int, s *goquery.Selection) {
		imgURL := handler.CoverImage(s)
		fmt.Printf("image url: %s\n", imgURL)
	})

	titleSel := config.TitleSelector
	titleS := doc.Find(titleSel)
	title := handler.Title(titleS)
	fmt.Printf("title: %s\n", title)

	authorSel := config.AuthorSelector
	author := handler.Author(doc.Find(authorSel))
	fmt.Printf("author: %s\n", author)

	summarySel := config.SummarySelector
	summary := handler.Summary(doc.Find(summarySel))
	fmt.Printf("summary: %s\n", summary)

	updateSel := config.UpdateSelector
	update := handler.Update(doc.Find(updateSel))
	fmt.Printf("更新时间: %s\n", update)

	chapterListSel := config.ChapterUrlSelector
	chapterList := doc.Find(chapterListSel)
	getChaptersContent(handler, config, chapterList)
}

func main() {
	catelogURL := novConfig.Config.CatelogUrl
	buffer, err := getUtf8HtmlBytesFromURL(novConfig, catelogURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	reader := bytes.NewReader(buffer)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		fmt.Println(err)
		return
	}

	getNovel(novConfig, novConfig.Config, doc)
}

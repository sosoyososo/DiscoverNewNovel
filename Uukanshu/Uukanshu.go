package Uukanshu

import (
	"net/url"

	"../Data"
	"../Encoding"
	"github.com/PuerkitoBio/goquery"
)

/*
Config 保存 Uukanshu 的配置信息
*/
type Config struct {
	Config Data.NovelConifg
}

/*
New 创建一个默认的 Uukanshu 配置
*/
func New(catelogURL string, fileStorePath string) Config {
	u, _ := url.Parse(catelogURL)
	baseURL := u.Scheme + "://" + u.Host + "/"
	c := Data.NovelConifg{"ASP.NET_SessionId=azxql35ktrk12lqlleegfrus; lastread=55516%3D0%3D; _ga=GA1.2.1243761825.1494000552; _gid=GA1.2.1926814091.1494322381; fcip=111",
		fileStorePath,
		catelogURL,
		baseURL,
		"dd > h1 > a",
		"dd > h3",
		"dd > h2 > a",
		"dd > .shijian",
		".jieshao > dt > a > img",
		"#chapterList > li",
		"div#contentbox"}
	return Config{Config: c}
}

/*
Title  获取从一个节点中获取 title
*/
func (u Config) Title(s *goquery.Selection) string {
	title := s.Text()
	return title
}

/*
Author 从节点中获取作者名字
*/
func (u Config) Author(s *goquery.Selection) string {
	text := s.Text()
	return text
}

/*
Summary 从节点中获取简介
*/
func (u Config) Summary(s *goquery.Selection) string {
	text := s.Text()
	return text
}

/*
Update 从节点中获取最新更新时间
*/
func (u Config) Update(s *goquery.Selection) string {
	text := s.Text()
	return text
}

/*
CoverImage 从节点中获取封面图片 URL
*/
func (u Config) CoverImage(s *goquery.Selection) string {
	url, isExist := s.Attr("src")
	if isExist {
		return url
	}
	return ""
}

/*
ChapterURL 从节点中获取章节页面的URL
*/
func (u Config) ChapterURL(s *goquery.Selection) string {
	aTag := s.Find("a")
	url, isExist := aTag.Attr("href")
	if isExist {
		return url
	}
	return ""
}

/*
ChapterTitle 从节点中获取章节页面的题目
*/
func (u Config) ChapterTitle(s *goquery.Selection) string {
	text := s.Text()
	return text
}

/*
ChapterContent 从页面中获取章节内容
*/
func (u Config) ChapterContent(s *goquery.Selection) string {
	text := s.Text()
	return text
}

/*
ConvertToUtf8 将返回的数据转换成UTF8
*/
func (u Config) ConvertToUtf8(s []byte) ([]byte, error) {
	buffer, err := Encoding.GbkToUtf8(s)
	return buffer, err
}

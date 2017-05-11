package Data

import "github.com/PuerkitoBio/goquery"

type NovelInfoHandler interface {
	Title(s *goquery.Selection) string
	Author(s *goquery.Selection) string
	Summary(s *goquery.Selection) string
	Update(s *goquery.Selection) string
	CoverImage(s *goquery.Selection) string
	ChapterURL(s *goquery.Selection) string
	ChapterTitle(s *goquery.Selection) string
	ChapterContent(s *goquery.Selection) string
	ConvertToUtf8(s []byte) ([]byte, error)
}

type NovelConifg struct {
	Cookie                 string
	SavePath               string
	CatelogUrl             string
	BaseUrl                string
	TitleSelector          string
	SummarySelector        string
	AuthorSelector         string
	UpdateSelector         string
	CoverImageSelector     string
	ChapterUrlSelector     string
	ChapterContentSelector string
}

type ChapterConfig struct {
	Index int
	S     *goquery.Selection
}

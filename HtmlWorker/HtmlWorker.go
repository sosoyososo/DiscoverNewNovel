package HtmlWorker

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

/*
目的：
	1.通过URL加载网页；
	2.根据selector获取html node列表
	3.调用操作处理node列表
使用步骤:
	1. 创建 Worker
	2. 对创建的 Worker 进行必要的配置
	3. 调用 Run
*/

/*
WorkerAction 封装一个对html文档特定内容的操作
*/
type WorkerAction struct {
	Selector string
	Action   func(selection *goquery.Selection)
}

/*
Worker 通过URL获取html文档，然后进行特定处理
*/
type Worker struct {
	URL         string
	Action      []WorkerAction
	CookieStrig string
	Encoder     func(s []byte) ([]byte, error)
	OnFail      func(err error)
}

/*
NewAction 创建一个 WorkerAction
*/
func NewAction(selector string, handler func(sel *goquery.Selection)) WorkerAction {
	a := WorkerAction{}
	a.Action = handler
	a.Selector = selector
	return a
}

/*
New 创建一个多操作 Worker
*/
func New(url string, action []WorkerAction) Worker {
	w := Worker{}
	w.Action = action
	w.URL = url
	return w
}

/*
SingleActionWorker 创建一个单操作 Worker
*/
func SingleActionWorker(url string, selector string, handler func(selection *goquery.Selection)) Worker {
	action := NewAction(selector, handler)
	worker := New(url, []WorkerAction{action})
	return worker
}

/*
Run 开始执行
*/
func (w *Worker) Run() {
	buffer, err := w.getUtf8HtmlBytesFromURL()
	if nil == err {
		w.doWork(buffer)
	} else {
		if nil != w.OnFail {
			w.OnFail(err)
		} else {
			fmt.Print(err)
		}
	}
}

func (w *Worker) getUtf8HtmlBytesFromURL() ([]byte, error) {
	// 校验 URL
	if len(w.URL) <= 0 {
		return []byte{}, errors.New("请求失败")
	}

	req, err := http.NewRequest("GET", w.URL, nil)
	if err != nil {
		fmt.Println(err)
		return []byte{}, err
	}

	if len(w.CookieStrig) > 0 {
		cookieList := strings.Split(w.CookieStrig, ";")
		for i := 0; i < len(cookieList); i++ {
			items := strings.Split(cookieList[i], "=")
			if len(items) >= 2 {
				cookie := http.Cookie{Name: items[0], Value: items[1]}
				req.AddCookie(&cookie)
			}
		}
	}
	tr := &http.Transport{
		DisableCompression: true,
	}

	var client = &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if strings.HasPrefix(resp.Status, "200") {
		buffer, err := ioutil.ReadAll(resp.Body)
		if len(buffer) <= 0 {
			return nil, err
		}
		if nil != w.Encoder {
			buffer, err = w.Encoder(buffer)
		}
		if len(buffer) <= 0 {
			return []byte{}, err
		}
		return buffer, nil
	}
	fmt.Println(resp.Status)
	return []byte{}, errors.New("请求失败")
}

func (w *Worker) doWork(buffer []byte) {
	reader := bytes.NewReader(buffer)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < len(w.Action); i++ {
		action := w.Action[i]
		action.Action(doc.Find(action.Selector))
	}
}

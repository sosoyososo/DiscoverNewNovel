package main

/*
	Author : Karsa
	Data : 2017-5-6
*/

/*
主体逻辑:
	2. 获取目录所在URL，根据规则分析出目录列表的标题和URL
	3. 读取各章节内容
	4. 分析网页内容提取征文信息
TODO:
	1. 保存整体信息，图片，标题，作者等
	2. 支持增量更新
		a. 将分析出来的章节列表保存到数据库中
		b. 标记已经下载好的章节
		c. 下次下载读取数据库，然后获取内容没有获取成功的章节，只获取没有获取成功的内容
*/
import (
	"fmt"

	"./Uukanshu"
)

func main() {
	// loadData()
	testSpider()
}

func testSpider() {
	ch := make(chan int, 1)
	Uukanshu.RunSpider(func() {
		ch <- 1
	})
	<-ch
	fmt.Println("结束")
}

func loadData() {
	ch := make(chan int, 1)
	Uukanshu.RunCateFetch("http://www.uukanshu.net/b/11356/", 3, func() {
		ch <- 1
	})
	<-ch
	fmt.Println("结束")
}

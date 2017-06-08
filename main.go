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

/*
发现小说的整体设计:
1. 从一个入口url获取网页内容，从内容中提取所有的连接，
2. 选择合适的再执行第一步
3. 在发现url的过城中，判断url如果是小说目录，就使用这个URL作为小说的入口，获取小说信息

问题:
1. 发现小说遍历网页的所有操作是异步的，过程中需要判断页面是否遍历过，小说是否存储过。后两者的操作是查询数据库获得的，需要跟前者进行同步
*/

/*
1.
*/
import (
	"fmt"
	"time"

	"./Uukanshu"
)

func main() {
	runNovelDiscover()
	// collectionNovelsInfo()
}

// 发现新的小说
// 开始后每24小时执行一次
func runNovelDiscover() {
	ch := make(chan int, 1)
	Uukanshu.RunSpider(func() {
		ch <- 1
	})
	<-ch
	fmt.Print("一次结束")
}

// 搜集每本小说的信息
func collectionNovelsInfo() {
	ch := make(chan int, 1)
	Uukanshu.CollecteNovelInfo(func() {
		ch <- 1
	})
	<-ch
	fmt.Print("结束")
}

// 发现新的章节
// 开始后每隔小时执行一次
func runChapterDiscovery() {
	Uukanshu.DiscoverNewChapters(func() {
		time.Sleep(time.Hour)
		runChapterDiscovery()
	})
}

# DiscoverNewNovel

> Used to find new novel from web site, and this is just the start and <b>not even close to the first milestone</b>. 

> Something happened here, MongoDB Collection support **no more than 16M** content,and it's easy to reach. I can use **GridFS** to get around this, which I think it's not so proper. Which I prefered is to just follow mongo guide ,and limit it's single collection content size. This needs addtional split and query support, and as a side project this will take a while for me .

## Milestone
1. - [ ] Just run through from discover new novel to find all chapters in this novel
2. - [ ] Use config file instead of hard coded config
3. - [ ] Run discover in back ground and run timely
4. - [ ] Limit Chapters Collection content size.

## Config the Env
1. [Config Go Env](https://golang.org/doc/install)
2. [Install MonogoDB and run](https://docs.mongodb.com/manual/installation/)
3. Install all go packages needed use ```go get  $packageNames```

## Used packages
```
github.com/PuerkitoBio/goquery
golang.org/x/text/encoding/simplifiedchinese
golang.org/x/text/transform
gopkg.in/mgo.v2/bson
gopkg.in/mgo.v2
```

## Other Config
本地开一个mongodb，所有爬取到的数据会存到这里。

# 数据库结构
所有的小说基本信息存储在名为novels的集合中。
另有一个总表存储每个小说的章节所在集合的名称。
其余所有的集合表用来存储章节列表。
package MongoDb

import (
	"strconv"
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ChapterCollection struct {
	URLs           []string //collectionName 对应的collection存储内容的小说列表
	CollectionName string
}

var (
	db                          *mgo.Database
	novelCollection             *mgo.Collection
	chapterCollectionCollection *mgo.Collection //保存章节表信息
	chapterCollections          []ChapterCollection
)

const (
	/*
		之前测试到一个collection中存储讲经16万章，
		假设每个小说都有2000章，每个collection可以存储80部小说
		在此之上加上翻倍的冗余，每个collection存储40部小说
	*/
	maxNovelCountInOneChapterCollection = 40
	dbAddress                           = "127.0.0.1:27017"
	dbName                              = "novel"
	novelCollectionName                 = "novels"
	chapterCCName                       = "chaptercollections"
)

func GetUukanshuNovelCollection() *mgo.Collection {
	createSessionIfNeed()
	return novelCollection
}

func GetUukanshuChapterCollection(url string) *mgo.Collection {
	createSessionIfNeed()

	for i := 0; i < len(chapterCollections); i++ {
		/*
			查询是否已经针对这个小说建表
		*/
		for j := 0; j < len(chapterCollections[i].URLs); j++ {
			if chapterCollections[i].URLs[j] == url {
				return db.C(chapterCollections[i].CollectionName)
			}
		}

		/*
			查询表存储内容是否达到上限
		*/
		if len(chapterCollections[i].URLs) < maxNovelCountInOneChapterCollection {
			urls := append(chapterCollections[i].URLs, url)
			chapterCollections[i].URLs = urls

			chapterName := chapterCollections[i].CollectionName
			err := db.C(chapterCCName).Update(bson.M{"collectionname": chapterName}, bson.M{"$set": bson.M{"urls": urls}})
			if nil != err {
				panic(err)
			}
			return db.C(chapterCollections[i].CollectionName)
		}
	}

	/*
		chapterCollections没有内容或者所有collection达到限
	*/
	chapterCollectionCount := len(chapterCollections)
	newName := strings.Join([]string{"chapter", strconv.Itoa(chapterCollectionCount + 1)}, "")

	chapterCollection := ChapterCollection{}
	chapterCollection.URLs = []string{url}
	chapterCollection.CollectionName = newName

	err := db.C(chapterCCName).Insert(&chapterCollection)
	if nil != err {
		panic(err)
	}

	chapterCollections = append(chapterCollections, chapterCollection)

	return db.C(newName)
}

func createSessionIfNeed() {
	if db == nil {
		dbSession, err := mgo.Dial(dbAddress)
		if err != nil {
			panic(err)
		}
		db = dbSession.DB(dbName)

		novelCollection = db.C(novelCollectionName)
		chapterCollectionCollection = db.C(chapterCCName)

		//加载章节列表collection名称表
		chapterCollections = []ChapterCollection{}
		iter := chapterCollectionCollection.Find(bson.M{}).Iter()
		defer iter.Close()
		chapterCollection := ChapterCollection{}
		for iter.Next(&chapterCollection) {
			chapterCollections = append(chapterCollections, chapterCollection)
		}
	}
}

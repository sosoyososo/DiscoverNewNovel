package MongoDb

import (
	"strconv"
	"strings"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type ChapterCollection struct {
	urls           []string
	collectionName string
}

var (
	db                          *mgo.Database
	novelCollection             *mgo.Collection
	chapterCollectionCollection *mgo.Collection
	chapterCollections          []ChapterCollection
)

const (
	// 之前测试到一个collection中存储讲经16万章，每个小说假设都有2000章，每个collection可以存储80部小说，在此之上加上翻倍的冗余，每个collection存储40部小说
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

func GetChapterCollection(url string) *mgo.Collection {
	createSessionIfNeed()
	for i := 0; i < len(chapterCollections); i++ {
		for j := 0; j < len(chapterCollections[i].urls); j++ {
			if chapterCollections[i].urls[j] == url {
				return db.C(chapterCollections[i].collectionName)
			}
		}
	}

	/*
		章节列表collection还不存在,有表不满
	*/
	for i := 0; i < len(chapterCollections); i++ {
		if len(chapterCollections[i].urls) < maxNovelCountInOneChapterCollection {
			urls := append(chapterCollections[i].urls, url)
			chapterCollections[i].urls = urls
			db.C(chapterCCName).Update(bson.M{"collectionName": chapterCollections[i]}, bson.M{"$set": bson.M{"urls": urls}})
			return db.C(chapterCollections[i].collectionName)
		}
	}

	/*
		章节列表collection还不存在,无表不满
	*/

	chapterCollectionCount := len(chapterCollections)
	newName := strings.Join([]string{"chapter", strconv.Itoa(chapterCollectionCount + 1)}, "")
	chapterCollection := ChapterCollection{}
	chapterCollection.urls = []string{url}
	chapterCollection.collectionName = newName

	db.C(chapterCCName).Insert(chapterCollection)
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

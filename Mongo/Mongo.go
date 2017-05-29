package Mongo

// var (
// 	DbAddress              = "127.0.0.1:2701"
// 	_session  *mgo.Session = nil
// )

/*
ConnectTo 连接到一个数据库，并且返回对应的Collectionlai进行操作
	1. createWhenNoExist如果为true，当访问的db或者collection不存在的时候就直接创建一个
	2. 会缓存创建的数据库连接，来下次使用

*/
// func ConnectTo(dbName string, collectionName string, createWhenNoExist bool) (*mgo.Collection, error) {
// 	if nil == _session {
// 		session, err := mgo.Dial(DbAddress)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	db := _session.DB(dbName)
// 	uuCollection := db.C(collectionName)
// 	return uuCollection, nil
// }

// func getCollection(db mgo.Database, collectionName string, createWhenNoExist bool) (*mgo.Collection, error) {
// }

var actionChan chan func()
var dbActionInRunning = false

func DBAction(action func()) {
	if actionChan == nil {
		actionChan = make(chan func(), 1024)
	}
	actionChan <- action
	if dbActionInRunning == false {
		go realDbAction()
	}
}

func realDbAction() {
	action := <-actionChan
	action()
}

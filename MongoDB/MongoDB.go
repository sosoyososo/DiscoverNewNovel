package MongoDB

import (
	"fmt"
	"log"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Person struct {
	Name  string
	Phone string
}

func TestMongo() {
	session, err := mgo.Dial("127.0.0.1:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	// session.SetMode(mgo.Monotonic, true)

	c := session.DB("test").C("test")
	p := Person{}
	p.Name = "Ale2"
	err = c.Insert(&p)
	if err != nil {
		log.Fatal(err)
	}

	result := Person{}
	err = c.Find(bson.M{"name": "Ale2"}).One(&result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Ale2:", result.Phone)
}

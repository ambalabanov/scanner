package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/lair-framework/go-nmap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	wg      sync.WaitGroup
	config  configuration
	nmapXML nmap.NmapRun
)

type configuration struct {
	Db struct {
		URI  string `json:"uri"`
		Db   string `json:"db"`
		Coll string `json:"coll"`
	} `json:"database"`
}

func init() {
	var err error
	fmt.Print("Load config.json...")
	config, err = loadJSON("config.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	//nmap --open -p- -i nmap_input -oX nmap_output.xml
	fmt.Print("Load nmap_output.xml...")
	nmapXML, err = loadXML("nmap_output.xml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Prepare database...")
	err = dbDelete(bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
}

func main() {
	log.Println("Start scan...")
	for _, h := range nmapXML.Hosts {
		for _, p := range h.Ports {
			fmt.Println(string(h.Hostnames[0].Name), int(p.PortId))
			go checkHTTP(string(h.Hostnames[0].Name), int(p.PortId))
		}
	}
	log.Printf("Active gorutines %v\n", runtime.NumGoroutine())
	wg.Wait()
	log.Println("Scan complete!")
	log.Println("Retrive data from db...")
	dbFind(bson.M{})

}

func loadJSON(filename string) (configuration, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return configuration{}, err
	}
	var c configuration
	err = json.Unmarshal(bytes, &c)
	if err != nil {
		return configuration{}, err
	}
	return c, nil
}

func loadXML(filename string) (nmap.NmapRun, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nmap.NmapRun{}, err
	}
	var x nmap.NmapRun
	err = xml.Unmarshal(bytes, &x)
	if err != nil {
		return nmap.NmapRun{}, err
	}
	return x, nil
}

func checkHTTP(host string, port int) {
	wg.Add(1)
	defer wg.Done()
	url := fmt.Sprintf("http://%s:%d", host, port)
	client := http.Client{
		Timeout: 1 * time.Second,
	}
	r, err := client.Head(url)
	if err != nil {
		return
	}
	if r.StatusCode != http.StatusNotFound {
		dbInsert(bson.M{"host": host, "port": port, "url": url, "status": r.Status, "header": r.Header})
	}
}

func dbInsert(data bson.M) {
	collection, err := dbConnect()
	if err != nil {
		log.Fatalln("Db not connected!")
	}
	collection.InsertOne(context.TODO(), data)
	if err != nil {
		log.Fatal(err)
	}
}
func dbDelete(filter bson.M) error {
	collection, err := dbConnect()
	if err != nil {
		return err
	}
	collection.DeleteMany(context.TODO(), filter)
	if err != nil {
		return err
	}
	return nil
}

func dbFind(filter bson.M) {
	collection, err := dbConnect()
	if err != nil {
		log.Fatalln("db not connected!")
	}
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Fatal("Error on Finding all the documents", err)
	}
	for cur.Next(context.TODO()) {
		var result bson.M
		err = cur.Decode(&result)
		if err != nil {
			log.Fatal("Error on Decoding the document", err)
		}
		fmt.Println(result)
	}
}

func dbConnect() (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(config.Db.URI))
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return nil, err
	}
	collection := client.Database(config.Db.Db).Collection(config.Db.Coll)
	return collection, nil
}

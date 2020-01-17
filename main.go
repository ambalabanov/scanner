package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/ambalabanov/go-nmap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	wg         sync.WaitGroup
	collection *mongo.Collection
	hosts      documents
)

type configuration struct {
	Db    database `json:"database"`
	Hosts []host   `json:"hosts"`
	Nmap  struct {
		Use  bool   `json:"use"`
		File string `json:"file"`
	} `json:"nmap"`
}
type host struct {
	Name  string `json:"name"`
	Ports []int  `json:"ports"`
}
type database struct {
	URI  string `json:"uri"`
	Db   string `json:"db"`
	Coll string `json:"coll"`
}
type document struct {
	Name   string      `bson:"name"`
	Port   int         `bson:"port"`
	URL    string      `bson:"url"`
	Method string      `bson:"method"`
	Scheme string      `bson:"scheme"`
	Host   string      `bson:"host"`
	Status int         `bson:"status"`
	Header http.Header `bson:"header"`
	Body   []byte      `bson:"body"`
}

type documents []document

func init() {
	fmt.Print("Load config.json...")
	var config configuration
	if err := config.Load("config.json"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")

	fmt.Print("Connect to mongodb...")
	var err error
	if collection, err = dbConnect(config.Db); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Drop collection...")
	if err := dbDrop(collection); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Load hosts")
	if err := hosts.Load(&config); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
}

func main() {
	fmt.Print("Init scan...")
	for _, host := range hosts {
		go host.Scan()
	}
	fmt.Println("OK!")
	fmt.Printf("Count ports %v\n", len(hosts))
	fmt.Printf("Active gorutines %v\n", runtime.NumGoroutine())
	fmt.Print("Complete scan...")
	time.Sleep(1 * time.Second)
	wg.Wait()
	fmt.Println("OK!")
	fmt.Print("Retrive data from database...")
	filter := bson.M{"status": bson.M{"$ne": ""}}
	var results documents
	if err := results.Read(collection, filter); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Println("Print ALL documents")
	fmt.Println("Count ", len(results))
	for _, r := range results {
		fmt.Println(r.Method, r.Scheme, r.Host, http.StatusText(r.Status), r.Header.Get("Content-Type"))
	}
	fmt.Println("Print ONE document")
	var result document
	filter = bson.M{"name": bson.M{"$eq": "getinside.cloud"}, "port": bson.M{"$lt": 1024}}
	if err := result.Read(collection, filter); err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Host)
	fmt.Println(string(result.Body))

}

func (c *configuration) Load(filename string) error {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, c); err != nil {
		return err
	}
	return nil
}
func (d *documents) Load(config *configuration) error {
	if config.Nmap.Use {
		fmt.Printf("(%s)...", config.Nmap.File)
		bytes, err := ioutil.ReadFile(config.Nmap.File)
		if err != nil {
			return err
		}
		nmapXML, err := nmap.Parse(bytes)
		if err != nil {
			return err
		}
		for _, n := range nmapXML.Hosts {
			var doc document
			for _, p := range n.Ports {
				doc.Name = string(n.Hostnames[0].Name)
				doc.Port = int(p.PortId)
				*d = append(*d, doc)
			}
		}
	} else {
		fmt.Print("(config.json)...")
		for _, n := range config.Hosts {
			var doc document
			for _, p := range n.Ports {
				doc.Name = n.Name
				doc.Port = p
				*d = append(*d, doc)
			}
		}
	}
	return nil
}

func (d document) Scan() error {
	wg.Add(1)
	defer wg.Done()
	url := fmt.Sprintf("http://%s:%d", d.Name, d.Port)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Get(url)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(r.Body)
	d.URL = url
	d.Method = r.Request.Method
	d.Scheme = r.Request.URL.Scheme
	d.Host = r.Request.Host
	d.Status = r.StatusCode
	d.Header = r.Header
	d.Body = body

	if err := d.Write(collection); err != nil {
		return err
	}
	return nil
}

func (d *document) Write(c *mongo.Collection) error {
	data, err := bson.Marshal(d)
	if err != nil {
		return err
	}
	_, err = c.InsertOne(context.TODO(), data)
	if err != nil {
		return err
	}
	return nil
}

func (d *documents) Write(c *mongo.Collection) error {
	docs := *d
	for _, doc := range docs {
		err := doc.Write(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *document) Read(c *mongo.Collection, f bson.M) error {
	if err := c.FindOne(context.Background(), f).Decode(&d); err != nil {
		return err
	}
	return nil
}

func (d *documents) Read(c *mongo.Collection, f bson.M) error {
	cursor, err := c.Find(context.TODO(), f)
	if err != nil {
		return err
	}
	for cursor.Next(context.TODO()) {
		var result document
		if err := cursor.Decode(&result); err != nil {
			return err
		}
		*d = append(*d, result)
	}
	return nil
}

func dbDelete(c *mongo.Collection, filter bson.M) error {
	if _, err := c.DeleteMany(context.TODO(), filter); err != nil {
		return err
	}
	return nil
}

func dbDrop(c *mongo.Collection) error {
	if err := c.Drop(context.TODO()); err != nil {
		return err
	}
	return nil
}

func dbConnect(d database) (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(d.URI))
	if err = client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return nil, err
	}
	return client.Database(d.Db).Collection(d.Coll), nil
}

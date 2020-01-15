package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	config     configuration
	nmapXML    nmap.NmapRun
	collection *mongo.Collection
	usenmap    bool
	hosts      []host
)

type configuration struct {
	Db    database `json:"database"`
	Hosts []host   `json:"hosts"`
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
	Method string      `bson:"method"`
	Host   string      `bson:"host"`
	Port   int         `bson:"port"`
	URL    string      `bson:"url"`
	Status int         `bson:"status"`
	Header http.Header `bson:"header"`
	Body   []byte      `bson:"body"`
}

func init() {
	var err error
	fmt.Print("Load 'config.json' file...")
	config, err = loadJSON("config.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	if _, err := os.Stat("nmap_output.xml"); os.IsNotExist(err) {
		fmt.Println("File 'nmap_output.xml' not found!")
		fmt.Println("Use hosts from 'config.json' file")
		hosts = config.Hosts
	} else {
		fmt.Print("Load 'nmap_output.xml' file...")
		nmapXML, err = loadXML("nmap_output.xml")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("OK!")
		for _, n := range nmapXML.Hosts {
			var h host
			h.Name = string(n.Hostnames[0].Name)
			for _, p := range n.Ports {
				h.Ports = append(h.Ports, int(p.PortId))
			}
			hosts = append(hosts, h)
		}
	}
	fmt.Print("Connect to mongodb...")
	collection, err = dbConnect(config.Db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Drop collection...")
	err = dbDrop(collection)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
}

func main() {
	log.Println("Start scan...")
	for _, h := range hosts {
		for _, p := range h.Ports {
			go checkHTTP(h.Name, p)
		}
	}

	fmt.Printf("Active gorutines %v\n", runtime.NumGoroutine())
	time.Sleep(1 * time.Second)
	wg.Wait()
	log.Println("Scan complete!")
	fmt.Print("Retrive data from database...")
	filter := bson.M{"status": bson.M{"$ne": ""}}
	result, err := dbFind(filter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Println("______________")
	fmt.Println("Print results:")
	fmt.Println("______________")
	for _, r := range result {
		fmt.Println(r.Host, r.Port, r.Header.Get("Server"))
	}
	for _, r := range result {
		fmt.Println(r.Method, r.URL, http.StatusText(r.Status), r.Header.Get("Content-Type"))
	}
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

func checkHTTP(host string, port int) error {
	wg.Add(1)
	defer wg.Done()
	url := fmt.Sprintf("http://%s:%d", host, port)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Get(url)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(r.Body)
	d := document{
		Method: r.Request.Method,
		Host:   host,
		Port:   port,
		URL:    url,
		Status: r.StatusCode,
		Header: r.Header,
		Body:   body,
	}
	err = d.Write(collection)
	if err != nil {
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

func dbDelete(filter bson.M) error {
	_, err := collection.DeleteMany(context.TODO(), filter)
	if err != nil {
		return err
	}
	return nil
}

func dbDrop(c *mongo.Collection) error {
	err := c.Drop(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func dbFind(filter bson.M) ([]*document, error) {
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return []*document{}, err
	}
	var results []*document
	for cursor.Next(context.TODO()) {
		var result document
		if err := cursor.Decode(&result); err != nil {
			return []*document{}, err
		}
		results = append(results, &result)
	}
	return results, nil
}

func dbConnect(d database) (*mongo.Collection, error) {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(d.URI))
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return nil, err
	}
	collection = client.Database(d.Db).Collection(d.Coll)
	return collection, nil
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ambalabanov/go-nmap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	wg    sync.WaitGroup
	db    database
	hosts documents
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
	URI        string `json:"uri"`
	Db         string `json:"db"`
	Coll       string `json:"coll"`
	Empty      bool   `json:"empty"`
	collection *mongo.Collection
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
	Links  []string    `bson:"links"`
	Title  string      `bson:"title"`
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
	db = config.Db
	if err := db.connect(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	if config.Db.Empty {
		fmt.Print("Drop collection...")
		if err := db.drop(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("OK!")
	}
	fmt.Print("Load hosts...")
	if err := hosts.Load(&config); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
}

func main() {
	fmt.Print("Scan...")
	hosts.Scan()
	fmt.Println("OK!")
	fmt.Print("Retrive scan results...")
	filter := bson.M{}
	var results documents
	if err := results.Read(db.collection, filter); err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Parse body...")
	results.Parse()
	fmt.Println("OK!")
	fmt.Print("Results: ")
	results = documents{}
	filter = bson.M{"body": bson.M{"$ne": nil}, "title": bson.M{"$ne": ""}}
	if err := results.Read(db.collection, filter); err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(results))
	for _, res := range results {
		fmt.Println(res.Method, res.URL, http.StatusText(res.Status), res.Header.Get("server"))
		fmt.Println(res.Title)
		for _, l := range res.Links {
			fmt.Println(l)
		}
	}
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
	for _, s := range []string{"http", "https"} {
		if config.Nmap.Use {
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
					doc.Scheme = s
					*d = append(*d, doc)
				}
			}
		} else {
			for _, n := range config.Hosts {
				var doc document
				for _, p := range n.Ports {
					doc.Name = n.Name
					doc.Port = p
					doc.Scheme = s
					*d = append(*d, doc)
				}
			}
		}
	}
	return nil
}

func (d document) Scan() error {
	defer wg.Done()
	url := fmt.Sprintf("%s://%s:%d", d.Scheme, d.Name, d.Port)
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Head(url)
	if err != nil {
		return err
	}
	d.URL = url
	d.Method = r.Request.Method
	d.Scheme = r.Request.URL.Scheme
	d.Host = r.Request.Host
	d.Status = r.StatusCode
	d.Header = r.Header
	if err := d.Write(db.collection); err != nil {
		return err
	}
	return nil
}

func (d *documents) Scan() error {
	for _, doc := range *d {
		wg.Add(1)
		go doc.Scan()
	}
	wg.Wait()
	return nil
}

func (d *documents) Parse() error {
	for _, doc := range *d {
		wg.Add(1)
		go doc.Parse()
	}
	wg.Wait()
	return nil
}

func (d document) Parse() error {
	defer wg.Done()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	r, err := client.Get(d.URL)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	d.Body = body
	d.parseLinks(ioutil.NopCloser(bytes.NewBuffer(body)))
	d.parseTitle(ioutil.NopCloser(bytes.NewBuffer(body)))
	d.Method = r.Request.Method
	if err := d.Write(db.collection); err != nil {
		return err
	}
	return nil
}

func (d *document) parseLinks(b io.Reader) {
	var links []string
	tokenizer := html.NewTokenizer(b)
	for tokenType := tokenizer.Next(); tokenType != html.ErrorToken; {
		token := tokenizer.Token()
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.A {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}
				}
			}
		}
		tokenType = tokenizer.Next()
	}
	d.Links = links
}

func (d *document) parseTitle(b io.Reader) {
	tokenizer := html.NewTokenizer(b)
	for tokenType := tokenizer.Next(); tokenType != html.ErrorToken; {
		token := tokenizer.Token()
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.Title {
				tokenType = tokenizer.Next()
				if tokenType == html.TextToken {
					d.Title = tokenizer.Token().Data
					break
				}
			}
		}
		tokenType = tokenizer.Next()
	}
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

func (d *database) delete(filter bson.M) error {
	if _, err := d.collection.DeleteMany(context.TODO(), filter); err != nil {
		return err
	}
	return nil
}

func (d *database) drop() error {
	if err := d.collection.Drop(context.TODO()); err != nil {
		return err
	}
	return nil
}

func (d *database) connect() error {
	client, _ := mongo.Connect(context.TODO(), options.Client().ApplyURI(d.URI))
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		return err
	}
	d.collection = client.Database(d.Db).Collection(d.Coll)
	return nil
}

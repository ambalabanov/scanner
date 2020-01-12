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
	"os/exec"
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
)

type configuration struct {
	Db struct {
		URI  string `json:"uri"`
		Db   string `json:"db"`
		Coll string `json:"coll"`
	} `json:"database"`
	NmapParams string `json:"nmap"`
}

func init() {
	var err error
	fmt.Print("Load config.json...")
	config, err = loadJSON("config.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	if _, err := os.Stat("nmap_output.xml"); os.IsNotExist(err) {
		log.Println("nmap_output.xml not found!")
		if _, err := os.Stat("nmap_input.txt"); !os.IsNotExist(err) {
			fmt.Print("Run nmap...")
			exec.Command("bash", "-c", string(config.NmapParams)).Run()
			fmt.Println("ОК!")
		} else {
			log.Fatal("nmap_input.txt not found!")
		}
	}
	fmt.Print("Load nmap_output.xml...")
	nmapXML, err = loadXML("nmap_output.xml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Connect to database...")
	err = dbConnect()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK!")
	fmt.Print("Drop collection...")
	err = dbDrop()
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
	filter := bson.M{}
	result, err := dbFind(filter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
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
		Timeout: 5 * time.Second,
	}
	r, err := client.Head(url)
	if err != nil {
		return
	}
	go dbInsert(bson.M{"host": host, "port": port, "url": url, "status": r.Status, "header": r.Header})
}

func dbInsert(data bson.M) error {
	wg.Add(1)
	defer wg.Done()
	_, err := collection.InsertOne(context.TODO(), data)
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

func dbDrop() error {
	err := collection.Drop(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

func dbFind(filter bson.M) ([]bson.M, error) {
	opts := options.Find()
	opts.SetShowRecordID(false)
	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		return []bson.M{}, err
	}
	var result []bson.M
	if err = cursor.All(context.TODO(), &result); err != nil {
		return []bson.M{}, err
	}
	return result, nil
}

func dbConnect() error {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(config.Db.URI))
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return err
	}
	collection = client.Database(config.Db.Db).Collection(config.Db.Coll)
	return nil
}

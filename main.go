package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/handlers"
	"github.com/gorilla/mux"
)

type configuration struct {
	Server struct {
		Port int `json:"port"`
	} `json:"server"`
	Database struct {
		URI   string `json:"uri"`
		Db    string `json:"db"`
		Coll  string `json:"coll"`
		Empty bool   `json:"empty"`
	} `json:"database"`
}

var configPath = flag.String("c", "config.json", "Path to config.json")
var config configuration

func init() {
	flag.Parse()
	if err := config.load(configPath); err != nil {
		log.Fatal(err)
	}
	err := dao.Connect(config.Database.URI, config.Database.Db, config.Database.Coll)
	if err != nil {
		log.Fatal(err)
	}
	if config.Database.Empty {
		if err := dao.Drop(); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	router := mux.NewRouter()
	sub := router.PathPrefix("/api/v1").Subrouter()
	sub.HandleFunc("/parse", handlers.CreateScan).Methods("POST")
	sub.HandleFunc("/url", handlers.CreateParse).Methods("POST")
	sub.HandleFunc("/parse/", handlers.GetAllParse).Methods("GET")
	sub.HandleFunc("/parse", handlers.GetUrlParse).Methods("GET").Queries("url", "{url}")
	sub.HandleFunc("/parse/{id:[0-9a-fA-F]+}", handlers.GetIdParse).Methods("GET")
	sub.HandleFunc("/parse", handlers.DeleteAllParse).Methods("DELETE")
	sub.HandleFunc("/parse/{id:[0-9a-fA-F]+}", handlers.DeleteOneParse).Methods("DELETE")
	log.Printf("Server starting on port %v...\n", config.Server.Port)
	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("localhost:%v", config.Server.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func (c *configuration) load(filename *string) error {
	log.Printf("Load config: %v", *configPath)
	bytes, err := ioutil.ReadFile(*filename)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, c); err != nil {
		return err
	}
	return nil
}

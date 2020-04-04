package main

import (
	"encoding/json"
	"github.com/ambalabanov/scanner/services"
	"log"
	"os"
)

func main() {
	res := services.Parse(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(res)
	if err != nil {
		log.Panic(err)
	}
}

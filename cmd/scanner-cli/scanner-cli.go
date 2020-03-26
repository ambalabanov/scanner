package main

import (
	"encoding/json"
	"github.com/ambalabanov/scanner/services"
	"log"
	"os"
)

func main() {
	dd := services.LoadD(os.Stdin)
	res := services.Parse(dd)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(res)
	if err != nil {
		log.Panic(err)
	}
}

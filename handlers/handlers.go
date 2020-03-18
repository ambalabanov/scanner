package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/models"
	"github.com/ambalabanov/scanner/services"
	"github.com/gorilla/mux"
)

func handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func CreateScan(w http.ResponseWriter, r *http.Request) {
	hosts := LoadHosts(r)
	go services.Parse(hosts)
	http.Error(w, "Scan was successfully created", http.StatusCreated)
}

func CreateParse(w http.ResponseWriter, r *http.Request) {
	var hosts models.Documents
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&hosts)
	if err != nil {
		handleError(w, err)
	}
	go services.Parse(hosts)
	http.Error(w, "Parse was successfully created", http.StatusCreated)
}

func GetAllParse(w http.ResponseWriter, _ *http.Request) {
	hosts, err := dao.FindAll()
	if err != nil {
		handleError(w, err)
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	err = JSONResponse(w, hosts)
	if err != nil {
		handleError(w, err)
	}
}

func GetOneParse(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	hosts, err := dao.FindOne(params["id"])
	if err != nil {
		handleError(w, err)
	}
	if len(hosts) == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	err = JSONResponse(w, hosts)
	if err != nil {
		handleError(w, err)
	}
}

func JSONResponse(w http.ResponseWriter, d []models.Document) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(d)
	return err
}

func DeleteOneParse(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	count, err := dao.DeleteOne(params["id"])
	if err != nil {
		handleError(w, err)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

func LoadHosts(r *http.Request) models.Documents {
	var dd models.Documents
	scanner := bufio.NewScanner(r.Body)
	for scanner.Scan() {
		var d models.Document
		for _, s := range []string{"http", "https"} {
			for _, p := range []int{80, 443, 8000, 8080, 8443} {
				d.Scheme = s
				d.URL = fmt.Sprintf("%s://%s:%d", s, scanner.Text(), p)
				dd = append(dd, d)
			}
		}
	}
	return dd
}

func DeleteAllParse(w http.ResponseWriter, _ *http.Request) {
	count, err := dao.DeleteAll()
	if err != nil {
		handleError(w, err)
	}
	if count == 0 {
		http.Error(w, "Document not found", http.StatusNotFound)
	}
}

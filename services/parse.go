package services

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	ports      = []int{80, 443, 8000, 8080, 8443}
	scheme     = []string{"http", "https"}
	regex      = `([xc]srf)|(token)`
	numWorkers = 100
)

func ParseH(dd models.Documents) {
	result := Parse(dd)
	dao.InsertMany(result)
}

func Parse(dd models.Documents) models.Documents {
	jobs := make(chan models.Document, len(dd))
	results := make(chan models.Document, len(dd))
	var res models.Documents
	for _, d := range dd {
		jobs <- d
	}
	for w := 1; w <= numWorkers; w++ {
		go workerParse(jobs, results)
	}
	close(jobs)
	for i := 0; i < len(dd); i++ {
		r := <-results
		if r.Status != 0 && r.Status != 400 {
			res = append(res, r)
		}
	}
	return res
}

func workerParse(jobs <-chan models.Document, results chan<- models.Document) {
	for d := range jobs {
		client := &http.Client{
			Timeout: 3 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		r, err := client.Get(d.URL)
		if err != nil {
			results <- d
			continue
		}
		d.Method = r.Request.Method
		d.Scheme = r.Request.URL.Scheme
		d.Host = r.Request.Host
		d.Status = r.StatusCode
		d.Header = r.Header
		body, err := ioutil.ReadAll(r.Body)
		d.CNAME = getCNAME(d.Domain)
		d.Subdomaintakeover = SubCheck(body)
		if err == nil {
			ParseBody(ioutil.NopCloser(bytes.NewBuffer(body)), &d)
		}
		d.UpdatedAt = time.Now()
		results <- d
	}
}

func ParseBody(b io.Reader, d *models.Document) {
	doc, err := goquery.NewDocumentFromReader(b)
	if err != nil {
		return
	}
	//parse links
	var links []string
	doc.Find("a").Each(func(i int, l *goquery.Selection) {
		href, exists := l.Attr("href")
		if exists {
			links = append(links, href)
		}
	})
	RemoveDuplicates(&links)
	d.Links = links
	//parse title
	t := doc.Find("title").First()
	d.Title = strings.TrimSpace(t.Text())
	//parse forms
	formsMap := make(map[*models.Form]bool)
	var formsSlice []models.Form
	doc.Find("form").Each(func(i int, s *goquery.Selection) {
		f := new(models.Form)
		if method, exists := s.Attr("method"); exists {
			f.Method = method
		}
		if action, exists := s.Attr("action"); exists {
			f.Action = action
		}
		s.Find("input").Each(func(i int, s *goquery.Selection) {
			input := new(models.Input)
			if n, exists := s.Attr("name"); exists {
				input.Name = n
				//find csrf token
				re := regexp.MustCompile(regex)
				if re.FindStringIndex(n) != nil {
					f.CSRF = true
				}
			}
			if t, exists := s.Attr("type"); exists {
				input.Type = t
			}
			if v, exists := s.Attr("value"); exists {
				input.Value = v
			}
			f.Input = append(f.Input, *input)
		})
		if !formsMap[f] {
			formsMap[f] = true
			formsSlice = append(formsSlice, *f)
		}
	})
	d.Forms = formsSlice
	//parse scripts
	var scripts []string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists {
			scripts = append(scripts, src)
		}
	})
	RemoveDuplicates(&scripts)
	d.Scripts = scripts
}

func RemoveDuplicates(input *[]string) {
	found := make(map[string]bool)
	var unique []string
	for _, val := range *input {
		if found[val] == false {
			found[val] = true
			unique = append(unique, val)
		}
	}
	*input = unique
}

func LoadD(r io.Reader) models.Documents {
	var dd models.Documents
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var d models.Document
		for _, s := range scheme {
			for _, p := range ports {
				d.ID = primitive.NewObjectID()
				d.CreatedAt = time.Now()
				d.Scheme = s
				d.Domain = scanner.Text()
				d.URL = fmt.Sprintf("%s://%s:%d", s, d.Domain, p)
				dd = append(dd, d)
			}
		}
	}
	return dd
}

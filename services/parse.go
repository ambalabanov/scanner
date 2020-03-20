package services

import (
	"bufio"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/ambalabanov/scanner/dao"
	"github.com/ambalabanov/scanner/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

func ParseH(dd models.Documents) {
	result := Parse(dd)
	dao.InsertMany(result)
}

func Parse(dd models.Documents) models.Documents {
	var wg sync.WaitGroup
	var result models.Documents
	res := make(chan models.Document, len(dd))
	for _, doc := range dd {
		wg.Add(1)
		go ParseD(doc, &wg, res)
	}
	wg.Wait()
	for i, l := 0, len(res); i < l; i++ {
		result = append(result, <-res)
	}
	return result
}

func ParseD(d models.Document, wg *sync.WaitGroup, res chan models.Document) {
	defer wg.Done()
	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	r, err := client.Get(d.URL)
	if err != nil {
		return
	}
	//400 The plain HTTP request was sent to HTTPS port
	if r.StatusCode == 400 {
		return
	}
	d.ID = primitive.NewObjectID()
	d.CreatedAt = time.Now()
	d.Method = r.Request.Method
	d.Scheme = r.Request.URL.Scheme
	d.Host = r.Request.Host
	d.Status = r.StatusCode
	d.Header = r.Header
	doc, _ := goquery.NewDocumentFromReader(r.Body)
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
	formsMap := make(map[string]bool)
	var formsSlice []models.Form
	doc.Find("form").Each(func(i int, s *goquery.Selection) {
		var f models.Form
		if method, exists := s.Attr("method"); exists {
			f.Method = method
		}
		if action, exists := s.Attr("action"); exists {
			f.Action = action
		}
		s.Find("input").Each(func(i int, s *goquery.Selection) {
			if name, exists := s.Attr("name"); exists {
				f.Input = append(f.Input, name)
				//find csrf token
				re := regexp.MustCompile(`([xc]srf)|(token)`)
				if re.FindStringIndex(name) != nil {
					f.CSRF = true
				}
			}
		})
		if formsMap[f.Action] == false {
			formsMap[f.Action] = true
			formsSlice = append(formsSlice, f)
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
	d.UpdatedAt = time.Now()
	res <- d
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

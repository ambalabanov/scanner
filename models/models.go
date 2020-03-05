package models

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"sync"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

//Document type
type Document struct {
	ID        primitive.ObjectID `bson:"_id"        json:"id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	URL       string             `bson:"url"        json:"url"`
	Method    string             `bson:"method"     json:"method"`
	Scheme    string             `bson:"scheme"     json:"scheme"`
	Host      string             `bson:"host"       json:"host"`
	Status    int                `bson:"status"     json:"status"`
	Header    http.Header        `bson:"header"     json:"header"`
	Body      []byte             `bson:"body"       json:"-"`
	Links     []string           `bson:"links"      json:"links"`
	Title     string             `bson:"title"      json:"title"`
	Forms     []string           `bson:"forms"      json:"forms"`
}

//Documents type
type Documents []Document

//Parse func
func (d *Documents) Parse() {
	var wg sync.WaitGroup
	var dd Documents
	wg.Wait()
	res := make(chan Document, len(*d))
	for _, doc := range *d {
		wg.Add(1)
		go doc.parse(res, &wg)
	}
	wg.Wait()
	for i, l := 0, len(res); i < l; i++ {
		dd = append(dd, <-res)
	}
	*d = dd
}

func (d Document) parse(res chan Document, wg *sync.WaitGroup) error {
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
	d.Method = r.Request.Method
	d.Scheme = r.Request.URL.Scheme
	d.Host = r.Request.Host
	d.Status = r.StatusCode
	d.Header = r.Header
	d.ID = primitive.NewObjectID()
	d.CreatedAt = time.Now()
	d.Body = body
	log.Println("Parse body")
	d.parseLinks(ioutil.NopCloser(bytes.NewBuffer(body)))
	d.parseTitle(ioutil.NopCloser(bytes.NewBuffer(body)))
	d.UpdatedAt = time.Now()
	res <- d
	return nil
}

func (d *Document) parseLinks(b io.Reader) {
	var links, forms []string
	tokenizer := html.NewTokenizer(b)
	for tokenType := tokenizer.Next(); tokenType != html.ErrorToken; {
		token := tokenizer.Token()
		if tokenType == html.StartTagToken {
			if token.DataAtom == atom.A || token.DataAtom == atom.Form {
				for _, attr := range token.Attr {
					switch attr.Key {
					case "href":
						links = append(links, attr.Val)
					case "action":
						forms = append(forms, attr.Val)
					}
				}
			}
		}
		tokenType = tokenizer.Next()
	}
	d.Links = links
	d.Forms = forms
}

func (d *Document) parseTitle(b io.Reader) {
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

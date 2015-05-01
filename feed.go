package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	"hawx.me/code/serve"

	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

var (
	port   = flag.String("port", "8080", "")
	socket = flag.String("socket", "", "")
)

const TINYLETTER = "http://tinyletter.com"

type Letters struct {
	letterPath string
	Title      string
	List       []Letter
}

type Letter struct {
	Title   string
	Href    string
	Desc    string
	PubDate time.Time
}

func (l Letters) WriteRss(w io.Writer) error {
	feed := &feeds.Feed{
		Title:   l.Title,
		Link:    &feeds.Link{Href: TINYLETTER + l.letterPath},
		Created: time.Now(),
	}

	for _, letter := range l.List {
		feed.Items = append(feed.Items, &feeds.Item{
			Title:       letter.Title,
			Link:        &feeds.Link{Href: letter.Href},
			Description: letter.Desc,
			Created:     letter.PubDate,
		})
	}

	return feed.WriteRss(w)
}

func get(letterPath string) (Letters, error) {
	url := TINYLETTER + path.Join(letterPath, "archive")
	log.Println("GET", url)

	resp, err := http.Get(url)
	if err != nil {
		return Letters{}, err
	}
	if resp.StatusCode != 200 {
		return Letters{}, errors.New(resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return Letters{}, err
	}

	letters := Letters{
		letterPath,
		strings.TrimSpace(doc.Find("title").Text()),
		[]Letter{},
	}

	doc.Find(".message-list li").Each(func(i int, s *goquery.Selection) {
		dateStr := s.Find(".message-date").Text()
		link, _ := s.Find(".message-link").Attr("href")
		title := s.Find(".message-link span").Text()
		desc := s.Find(".message-snippet").Text()

		date, err := time.Parse("January 02, 2006", strings.TrimSpace(dateStr))
		if err != nil {
			log.Println(err)
			date = time.Now()
		}

		letters.List = append(letters.List, Letter{
			strings.TrimSpace(title),
			link,
			strings.TrimSpace(desc),
			date,
		})
	})

	return letters, nil
}

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		letters, err := get(r.URL.Path)
		if err != nil {
			log.Println(err)
			w.WriteHeader(400)
			return
		}

		w.Header().Add("Content-Type", "application/rss+xml")
		err = letters.WriteRss(w)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
	})

	serve.Serve(*port, *socket, http.DefaultServeMux)
}

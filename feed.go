package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	"github.com/hawx/serve"

	"flag"
	"fmt"
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
	Title string
	List  []Letter
}

type Letter struct {
	Title   string
	Href    string
	Desc    string
	PubDate time.Time
}

func get(letterPath string) (Letters, error) {
	url := TINYLETTER + path.Join(letterPath, "archive")
	log.Println("GET", url)

	resp, err := http.Get(url)
	if err != nil {
		return Letters{}, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return Letters{}, err
	}

	letters := Letters{
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
			w.WriteHeader(500)
			return
		}

		feed := &feeds.Feed{
			Title:   letters.Title,
			Link:    &feeds.Link{Href: TINYLETTER + r.URL.Path},
			Created: time.Now(),
		}

		for _, letter := range letters.List {
			feed.Items = append(feed.Items, &feeds.Item{
				Title:       letter.Title,
				Link:        &feeds.Link{Href: letter.Href},
				Description: letter.Desc,
				Created:     letter.PubDate,
			})
		}

		rss, err := feed.ToRss()
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}

		w.Header().Add("Content-Type", "application/rss+xml")
		fmt.Fprintf(w, rss)
	})

	serve.Serve(*port, *socket, http.DefaultServeMux)
}

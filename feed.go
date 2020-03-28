package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	"hawx.me/code/serve"

	"errors"
	"flag"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

type tinyletterClient struct {
	http    *http.Client
	baseURL string
}

func (c *tinyletterClient) get(letterPath string) (*feeds.Feed, error) {
	url := c.baseURL + path.Join(letterPath, "archive")
	log.Println("GET", url)

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	feed := &feeds.Feed{
		Title:   strings.TrimSpace(doc.Find("title").Text()),
		Link:    &feeds.Link{Href: c.baseURL + letterPath},
		Created: time.Now(),
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

		feed.Items = append(feed.Items, &feeds.Item{
			Title:       strings.TrimSpace(title),
			Link:        &feeds.Link{Href: link},
			Description: strings.TrimSpace(desc),
			Created:     date,
		})
	})

	return feed, nil
}

func main() {
	var (
		port   = flag.String("port", "8080", "")
		socket = flag.String("socket", "", "")
	)
	flag.Parse()

	client := &tinyletterClient{
		baseURL: "http://tinyletter.com",
		http:    http.DefaultClient,
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		letters, err := client.get(r.URL.Path)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusBadGateway)
			return
		}

		w.Header().Add("Content-Type", "application/rss+xml")
		err = letters.WriteRss(w)
		if err != nil {
			log.Println(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://tinyletter.com/favicon.ico", 301)
	})

	serve.Serve(*port, *socket, http.DefaultServeMux)
}

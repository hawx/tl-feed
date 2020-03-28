package main

import (
	"github.com/gorilla/feeds"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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
	defer resp.Body.Close()

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	feed := &feeds.Feed{
		Title:   htmlText(htmlFind(root, atom.Title)),
		Link:    &feeds.Link{Href: c.baseURL + letterPath},
		Created: time.Now(),
	}

	messageList := htmlFind(root, atom.Ul)
	if messageList == nil {
		return feed, nil
	}

	for li := messageList.FirstChild; li != nil; li = li.NextSibling {
		if li.DataAtom != atom.Li {
			continue
		}

		item := &feeds.Item{}

		for child := li.FirstChild; child != nil; child = child.NextSibling {
			switch htmlAttr(child, "class") {
			case "message-date":
				date, err := time.Parse("January 02, 2006", htmlText(child))
				if err != nil {
					log.Println(err)
					date = time.Now()
				}

				item.Created = date

			case "message-link":
				item.Title = htmlText(child)
				item.Link = &feeds.Link{Href: htmlAttr(child, "href")}

			case "message-snippet":
				item.Description = htmlText(child)
			}
		}

		feed.Items = append(feed.Items, item)
	}

	return feed, nil
}

func htmlFind(node *html.Node, a atom.Atom) *html.Node {
	if node.DataAtom == a {
		return node
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if result := htmlFind(child, a); result != nil {
			return result
		}
	}

	return nil
}

func htmlAttr(node *html.Node, attrName string) string {
	if node == nil {
		return ""
	}

	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}

	return ""
}

func htmlText(node *html.Node) string {
	if node == nil {
		return ""
	}

	if node.Type == html.TextNode {
		return node.Data
	}

	var parts []string

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		parts = append(parts, htmlText(c))
	}

	return strings.TrimSpace(strings.Join(parts, " "))
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

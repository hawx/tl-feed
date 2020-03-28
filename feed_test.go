package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hawx.me/code/assert"
)

func TestTinyletterClientGet(t *testing.T) {
	assert := assert.New(t)

	s := httptest.NewServer(http.FileServer(http.Dir("testdata")))

	client := &tinyletterClient{
		http:    http.DefaultClient,
		baseURL: s.URL,
	}

	feed, err := client.get("/tcarmody")
	assert.Nil(err)
	assert.Equal("Backlight", feed.Title)
	assert.Equal(s.URL+"/tcarmody", feed.Link.Href)
	assert.WithinDuration(time.Now(), feed.Created, time.Second)

	if assert.Len(feed.Items, 10) {
		item := feed.Items[2]
		assert.Equal("University dreams", item.Title)
		assert.Equal("*Hogwarts was the first and best home he had ever known. He and Voldemort and Snape, the abandoned boys, had all found home here.", item.Description)
		assert.Equal(time.Date(2018, time.September, 21, 0, 0, 0, 0, time.UTC), item.Created)
	}
}

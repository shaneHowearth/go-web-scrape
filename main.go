package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

const MAXSCRAPERS = 3
const DOMAIN = "https://www.stuff.co.nz"

var EMPTY struct{}

func main() {
	found := make(chan []string, MAXSCRAPERS*2)
	todo := make(map[string]struct{})
	done := make(map[string]struct{})
	// NOTE: The size of this buffer/channel needs to be able to hold all the possible urls
	// If the channel is too small, then the scrapers will block trying to send to it, and the main thread will not be able to clear the channel as it will be waiting for the worker goroutines to  signal that they have completed their task.
	url := make(chan string, 10000)
	signal := make(chan struct{}, MAXSCRAPERS)
	alive := 0

	for i := 0; i < MAXSCRAPERS; i++ {
		// Pool workers
		go fetch(url, found, signal)
	}

	alive += 1
	start := DOMAIN + "/"
	todo[start] = EMPTY
	url <- start
	done[start] = EMPTY

	for {
		// wait until scrapers signal that they have finished
		<-signal
		alive -= 1
		tmp := <-found
		for _, link := range tmp {
			if _, ok := todo[link]; !ok {
				todo[link] = EMPTY
				if _, ok := done[link]; !ok {
					alive += 1
					url <- DOMAIN + link
					done[link] = EMPTY
				}
			}
		}
		fmt.Printf("Found %d links\n", len(done))
	}

}

// fetch the requested webpage
func fetch(url <-chan string, found chan []string, signal chan struct{}) {
	var wg sync.WaitGroup
	for u := range url {
		fmt.Printf("Inside len(url) %d\n", len(url))
		fmt.Printf("Fetching %s\n", u)
		resp, err := http.Get(u)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()

		var buf bytes.Buffer
		tee := io.TeeReader(resp.Body, &buf)

		// Save the webpage to a file
		z := html.NewTokenizer(tee)

		wg.Add(1)
		go saveToFile(buf, u, &wg)

		// find all the links
		var links []string
		for {
			tt := z.Next()

			switch tt {
			case html.ErrorToken:
				found <- links
				signal <- EMPTY
				goto next
			case html.StartTagToken, html.EndTagToken:
				token := z.Token()
				if token.DataAtom.String() == "a" {
					for _, attr := range token.Attr {
						if attr.Key == "href" {
							link := attr.Val
							// Drop links leaving the site
							matched, _ := regexp.MatchString("http://.*", link)
							matcheds, _ := regexp.MatchString("https://.*", link)
							// Drop the javascript links
							matchedj, _ := regexp.MatchString(".*javascript.*", link)
							matchedm, _ := regexp.MatchString(".*mailto.*", link)
							if !matched && !matcheds && !matchedj && !matchedm {
								links = append(links, link)
							}
						}
					}
				}
			}
		}
	next:
		// Send found links back to master thread
		found <- links
		// Inform master thread that this goroutine has finished
		signal <- EMPTY
	}
	wg.Wait()
}

func saveToFile(page bytes.Buffer, filename string, wg *sync.WaitGroup) {
	re := regexp.MustCompile("/")
	filename = re.ReplaceAllString(filename, "-")
	ioutil.WriteFile("/tmp/"+filename, page.Bytes(), 0644)
	wg.Done()
}

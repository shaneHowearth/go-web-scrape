package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var maxScrapers int
var domain string
var saveTo string
var save bool
var delay int

var EMPTY struct{}

func main() {
	setOptions()
	checkOptions(maxScrapers)
	found := make(chan []string, maxScrapers*2)
	todo := make(map[string]struct{})
	done := make(map[string]struct{})
	// NOTE: The size of this buffer/channel needs to be able to hold all the possible urls
	// If the channel is too small, then the scrapers will block trying to send to it, and the main thread will not be able to clear the channel as it will be waiting for the worker goroutines to  signal that they have completed their task.
	url := make(chan string, 10000)
	signal := make(chan struct{}, maxScrapers)
	alive := 0

	for i := 0; i < maxScrapers; i++ {
		// Pool workers
		go fetch(url, found, signal)
	}

	alive += 1
	start := domain + "/"
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
					url <- domain + link
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
		time.Sleep(time.Duration(delay) * time.Millisecond)
		resp, err := http.Get(u)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()

		var buf bytes.Buffer
		tee := io.TeeReader(resp.Body, &buf)

		// Save the webpage to a file
		z := html.NewTokenizer(tee)

		if save {
			wg.Add(1)
			go saveToFile(buf, u, &wg)
		}
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
								link = strings.TrimPrefix(link, "www.stuff.co.nz")
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
	// Wait for any file save goroutines to complete before exiting
	if save {
		wg.Wait()
	}
}

func saveToFile(page bytes.Buffer, filename string, wg *sync.WaitGroup) {
	re := regexp.MustCompile("/")
	filename = re.ReplaceAllString(filename, "-")
	ioutil.WriteFile(saveTo+"/"+filename, page.Bytes(), 0644)
	wg.Done()
}

func setOptions() {
	// Set the options from commandline parameters (if supplied)
	defer flag.Parse()
	// Max scrapers
	flag.IntVar(&maxScrapers, "max-scrapers", 3, "Set the maximum number of scraper threads to be running, default: 3.")
	// Domain
	flag.StringVar(&domain, "domain", "www.google.com", "Domain that will be scraped.")
	// Save pages
	flag.BoolVar(&save, "save", true, "Save the downloaded pages.")
	// Directory to save webpages in
	flag.StringVar(&saveTo, "directory", "/tmp", "Directory that downloaded pages will be saved to, default is /tmp.")
	// Nice, delay each search by n seconds so the domain being scraped isn't slammed
	flag.IntVar(&delay, "delay", 0, "Delay each webpage fetch by n seconds (so you're not slamming the site).")
}

func checkOptions(maxScrapers int) {
	stop := false
	if maxScrapers == 0 {
		fmt.Println("Cannot have zero scrapers, nothing will be done!")
		stop = true
	}
	if stop {
		fmt.Println("Exiting...")
		os.Exit(0)
	}
}

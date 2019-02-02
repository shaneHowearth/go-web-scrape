package main

import (
	"fmt"
	"net/http"

	"golang.org/x/net/html"
)

func main() {
	fmt.Println(fetch("https://tour.golang.org/welcome/1"))
	fmt.Println(fetch("https://home.nzcity.co.nz/horoscope/reading.aspx?sign=Virgo"))
}

// fetch the requested webpage
func fetch(url string) []string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)
	// find all the links
	var links []string
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			//todo: links list shoudn't contain duplicates
			return links
		case html.StartTagToken, html.EndTagToken:
			token := z.Token()
			if token.DataAtom.String() == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						links = append(links, attr.Val)
					}

				}
			}
		}
	}
	return links
}

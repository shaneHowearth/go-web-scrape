# Multi threaded recursive web scraper #

## Requirements
Go

## Build and run application
Clone the repository, then build the application with `go build`

The binary `webscrape` can be execute with the following options:
* max-scrapers
	* Number of fetch threads running concurrently
* domain
	* Domain you wish the scraper to back up
* save
	* Save pages or not
* directory
	* Directory that pages will be saved to
* delay
	* Delay the fetch of each page by n seconds

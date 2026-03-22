package scraper

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gocolly/colly"
)

const startURL = "https://urnik.fov.um.si/Program/calendars/"

var CalendarLinks map[string]string

func GetCalendarLinks() {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		fmt.Println("invalid start URL:", err)
		return
	}

	collector := colly.NewCollector(
		colly.AllowedDomains(
			parsedURL.Host,
		),
	)

	collector.OnHTML("a[href]", func(element *colly.HTMLElement) {
		link := element.Attr("href")
		if link == "" {
			return
		}

		hrefURL, err := url.Parse(link)
		if err != nil {
			return
		}

		resolvedURL := parsedURL.ResolveReference(hrefURL)

		if resolvedURL.Host != parsedURL.Host {
			return
		}

		if !strings.HasPrefix(resolvedURL.Path, "/Program/calendars/") {
			return
		}

		if strings.TrimSpace(element.Text) == "" {
			return
		}

		CalendarLinks[element.Text] = resolvedURL.String()
		fmt.Printf("Calendar link found: %q -> %s\n", element.Text, resolvedURL.String())
	})

	collector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	collector.OnError(func(r *colly.Response, err error) {
		if r != nil && r.Request != nil {
			fmt.Printf("Error while visiting %s: %v\n", r.Request.URL.String(), err)
			return
		}
		fmt.Println("Collector error:", err)
	})

	collector.Visit(startURL)

}

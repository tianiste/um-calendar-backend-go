package scraper

import (
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"um-calendar-backend/internal/models"

	"github.com/gocolly/colly"
)

const startURL = "https://urnik.fov.um.si/Program/calendars/"

var CalendarLinks map[string]string

func fetchCalendarLinks() (map[string]string, error) {
	calendarLinks := make(map[string]string)

	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
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

		if !strings.HasSuffix(strings.ToLower(resolvedURL.Path), ".ics") {
			return
		}

		fullName := strings.TrimPrefix(resolvedURL.Path, "/Program/calendars/")
		fullName = strings.TrimSpace(fullName)
		if fullName == "" {
			return
		}

		calendarLinks[fullName] = resolvedURL.String()
		slog.Debug("calendar link found", "name", fullName, "url", resolvedURL.String())
	})

	collector.OnRequest(func(r *colly.Request) {
		slog.Debug("scraper visiting url", "url", r.URL.String())
	})

	collector.OnError(func(r *colly.Response, err error) {
		if r != nil && r.Request != nil {
			slog.Error("scraper request failed", "url", r.Request.URL.String(), "error", err)
			return
		}
		slog.Error("scraper collector error", "error", err)
	})

	if err := collector.Visit(startURL); err != nil {
		return nil, err
	}

	return calendarLinks, nil
}

func GetCalendarLinks() {
	links, err := fetchCalendarLinks()
	if err != nil {
		slog.Error("scraper failed to fetch calendar links", "error", err)
		return
	}

	if CalendarLinks == nil {
		CalendarLinks = make(map[string]string)
	}

	for key := range CalendarLinks {
		delete(CalendarLinks, key)
	}

	for key, value := range links {
		CalendarLinks[key] = value
	}
}

func FetchCalendars() ([]models.Calendar, error) {
	links, err := fetchCalendarLinks()
	if err != nil {
		return nil, err
	}

	calendars := make([]models.Calendar, 0, len(links))
	for name, link := range links {
		code := ""
		parts := strings.SplitN(name, "---", 2)
		if len(parts) > 1 {
			code = strings.TrimSpace(parts[0])
		}
		if code == "" {
			continue
		}

		calendars = append(calendars, models.Calendar{
			Code:    code,
			Name:    name,
			ICS_url: link,
		})
	}

	sort.Slice(calendars, func(i, j int) bool {
		left := parseCode(calendars[i].Code)
		right := parseCode(calendars[j].Code)

		if left != right {
			return left < right
		}

		return calendars[i].Name < calendars[j].Name
	})

	return calendars, nil
}

func parseCode(code string) int {
	trimmed := strings.TrimSpace(strings.TrimLeft(code, "0"))
	if trimmed == "" {
		if strings.TrimSpace(code) == "" {
			return int(^uint(0) >> 1)
		}
		return 0
	}

	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return int(^uint(0) >> 1)
	}

	return value
}

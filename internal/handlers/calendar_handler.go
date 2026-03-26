package handlers

import (
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	"um-calendar-backend/internal/cache"
	"um-calendar-backend/internal/repo"
	"um-calendar-backend/internal/scraper"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	calendarRepo *repo.CalendarRepo
	redisCache   *cache.RedisCache
	cacheTTL     time.Duration
	httpClient   *http.Client
}

func New(calendarRepo *repo.CalendarRepo, redisCache *cache.RedisCache, cacheTTL time.Duration) *Handler {
	return &Handler{
		calendarRepo: calendarRepo,
		redisCache:   redisCache,
		cacheTTL:     cacheTTL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (handler *Handler) HealthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (handler *Handler) ServeCalendarNames(ctx *gin.Context) {
	names := make([]string, 0)

	if handler.calendarRepo != nil {
		calendarsFromDB, err := handler.calendarRepo.GetAllCalendars()
		if err == nil {
			names = make([]string, 0, len(calendarsFromDB))
			for _, item := range calendarsFromDB {
				names = append(names, item.Name)
			}
			sort.Strings(names)
			ctx.JSON(http.StatusOK, names)
			return
		}

		log.Printf("failed to read calendar names from db: %v", err)
	}

	names = make([]string, 0, len(scraper.CalendarLinks))
	for key := range scraper.CalendarLinks {
		names = append(names, key)
	}
	sort.Strings(names)

	ctx.JSON(http.StatusOK, names)
}

func (handler *Handler) ServeCalendarICSByName(ctx *gin.Context) {
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		ctx.Status(http.StatusNotFound)
		return
	}

	calendarURL := handler.resolveCalendarLinkByName(name)
	if calendarURL == "" {
		ctx.Status(http.StatusNotFound)
		return
	}

	cacheKey := "calendar:ics:" + name
	if handler.redisCache != nil {
		cachedContent, found, err := handler.redisCache.Get(ctx.Request.Context(), cacheKey)
		if err != nil {
			log.Printf("redis get failed: %v", err)
		} else if found {
			ctx.Header("X-Cache", "HIT")
			ctx.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(cachedContent))
			return
		}
	}

	request, err := http.NewRequest(http.MethodGet, calendarURL, nil)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	response, err := handler.httpClient.Do(request)
	if err != nil {
		ctx.Status(http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		ctx.Status(http.StatusNotFound)
		return
	}

	fileContent, err := io.ReadAll(response.Body)
	if err != nil {
		ctx.Status(http.StatusBadGateway)
		return
	}

	if handler.redisCache != nil {
		if err := handler.redisCache.Set(ctx.Request.Context(), cacheKey, string(fileContent), handler.cacheTTL); err != nil {
			log.Printf("redis set failed: %v", err)
		}
		ctx.Header("X-Cache", "MISS")
	}

	ctx.Data(http.StatusOK, "text/calendar; charset=utf-8", fileContent)
}

func (handler *Handler) resolveCalendarLinkByName(name string) string {
	if handler.calendarRepo != nil {
		item, err := handler.calendarRepo.GetCalendarByName(name)
		if err == nil && item != nil {
			return item.ICS_url
		}

		if err != nil {
			log.Printf("failed to read single calendar by name from db: %v", err)
		}
	}

	if link, ok := scraper.CalendarLinks[name]; ok {
		return link
	}

	return ""
}

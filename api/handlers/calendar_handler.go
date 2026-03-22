package handlers

import (
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"um-calendar-backend/internal/repo"
	"um-calendar-backend/internal/scraper"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	calendarRepo *repo.CalendarRepo
}

type calendarResponse struct {
	Naziv string
	Link  string
}

func New(calendarRepo *repo.CalendarRepo) *Handler {
	return &Handler{calendarRepo: calendarRepo}
}

func (handler *Handler) HealthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (handler *Handler) ServeCalendarLinks(ctx *gin.Context) {
	if handler.calendarRepo != nil {
		calendarsFromDB, err := handler.calendarRepo.GetAllCalendars()
		if err == nil {
			output := make([]calendarResponse, 0, len(calendarsFromDB))
			for _, item := range calendarsFromDB {
				output = append(output, calendarResponse{Naziv: item.Name, Link: item.ICS_url})
			}

			sortCalendars(output)
			ctx.JSON(http.StatusOK, gin.H{
				"items": output,
				"count": len(output),
			})
			return
		}

		log.Printf("failed to read calendars from db: %v", err)
	}

	output := make([]calendarResponse, 0, len(scraper.CalendarLinks))
	for key, value := range scraper.CalendarLinks {
		output = append(output, calendarResponse{Naziv: key, Link: value})
	}

	sortCalendars(output)
	ctx.JSON(http.StatusOK, gin.H{
		"items": output,
		"count": len(output),
	})
}

func (handler *Handler) ServeSingleCalendar(ctx *gin.Context) {
	value := strings.TrimSpace(ctx.Param("value"))
	if value == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "missing calendar value",
		})
		return
	}

	code := extractCode(value)
	if code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid calendar value",
		})
		return
	}

	if handler.calendarRepo != nil {
		item, err := handler.calendarRepo.GetSingleCalendar(code)
		if err == nil && item != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"item": calendarResponse{Naziv: item.Name, Link: item.ICS_url},
			})
			return
		}

		if err != nil {
			log.Printf("failed to read single calendar from db: %v", err)
		}
	}

	for name, link := range scraper.CalendarLinks {
		if extractCode(name) == code {
			ctx.JSON(http.StatusOK, gin.H{
				"item": calendarResponse{Naziv: name, Link: link},
			})
			return
		}
	}

	ctx.JSON(http.StatusNotFound, gin.H{
		"error": "calendar not found",
	})
}

func extractCode(value string) string {
	parts := strings.SplitN(value, "---", 2)
	code := strings.TrimSpace(parts[0])
	if code == "" {
		return ""
	}
	return code
}

func sortCalendars(output []calendarResponse) {
	sort.Slice(output, func(i, j int) bool {
		left := calendarOrder(output[i].Naziv)
		right := calendarOrder(output[j].Naziv)

		if left != right {
			return left < right
		}

		return output[i].Naziv < output[j].Naziv
	})
}

func calendarOrder(name string) int {
	prefix := strings.SplitN(name, "---", 2)[0]
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return math.MaxInt
	}

	prefix = strings.TrimLeft(prefix, "0")
	if prefix == "" {
		return 0
	}

	value, err := strconv.Atoi(prefix)
	if err != nil {
		return math.MaxInt
	}

	return value
}

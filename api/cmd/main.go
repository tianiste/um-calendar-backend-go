package main

import (
	"net/http"
	"um-calendar-backend/internal/scraper"

	"github.com/gin-gonic/gin"
)

func main() {
	scraper.CalendarLinks = make(map[string]string)
	scraper.GetCalendarLinks()

	router := gin.Default()
	router.GET("/health", healthCheck)
	router.GET("/calendars", serveCalendarLinks)
	router.Run(":8080")
}

func healthCheck(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

type calendar struct {
	Naziv string
	Link  string
}

func serveCalendarLinks(ctx *gin.Context) {
	var output []calendar
	for key, value := range scraper.CalendarLinks {
		output = append(output, calendar{Naziv: key, Link: value})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"items": output,
		"count": len(output),
	})
}

package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"um-calendar-backend/internal/repo"
	"um-calendar-backend/internal/scraper"
)

type CalendarSyncService struct {
	repo       *repo.CalendarRepo
	httpClient *http.Client
}

func NewCalendarSyncService(calendarRepo *repo.CalendarRepo) *CalendarSyncService {
	return &CalendarSyncService{
		repo: calendarRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (service *CalendarSyncService) SyncCalendars() error {
	scrapedCalendars, err := scraper.FetchCalendars()
	if err != nil {
		return fmt.Errorf("scrape calendars: %w", err)
	}

	if err := service.repo.UpdateCalendars(scrapedCalendars); err != nil {
		return fmt.Errorf("upsert calendars: %w", err)
	}

	storedCalendars, err := service.repo.ListCalendarsForSync()
	if err != nil {
		return fmt.Errorf("list calendars: %w", err)
	}

	for _, calendar := range storedCalendars {
		err := service.checkAndUpdateCalendar(calendar.ID, calendar.ICS_url, calendar.ETag, calendar.LastModified, calendar.ContentHash)
		if err != nil {
			fmt.Printf("sync check failed for calendar %s (%s): %v\n", calendar.Code, calendar.Name, err)
		}
	}

	return nil
}

func (service *CalendarSyncService) StartHourly() {
	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			if err := service.SyncCalendars(); err != nil {
				fmt.Println("hourly calendar sync failed:", err)
			}
		}
	}()
}

func (service *CalendarSyncService) checkAndUpdateCalendar(calendarID int, calendarURL string, etag, lastModified, previousHash *string) error {
	req, err := http.NewRequest(http.MethodGet, calendarURL, nil)
	if err != nil {
		return err
	}

	if etag != nil && *etag != "" {
		req.Header.Set("If-None-Match", *etag)
	}
	if lastModified != nil && *lastModified != "" {
		req.Header.Set("If-Modified-Since", *lastModified)
	}

	response, err := service.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	now := time.Now().UTC()
	newETag := normalizedHeaderValue(response.Header.Get("ETag"))
	newLastModified := normalizedHeaderValue(response.Header.Get("Last-Modified"))

	if response.StatusCode == http.StatusNotModified {
		return service.repo.UpdateCalendarSyncState(calendarID, newETag, newLastModified, previousHash, now, false)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(bodyBytes)
	hashValue := hex.EncodeToString(hash[:])
	hasChanged := previousHash == nil || *previousHash != hashValue

	return service.repo.UpdateCalendarSyncState(calendarID, newETag, newLastModified, &hashValue, now, hasChanged)
}

func normalizedHeaderValue(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

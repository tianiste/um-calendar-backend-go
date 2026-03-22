package repo

import (
	"database/sql"
	"fmt"
	"time"
	"um-calendar-backend/internal/models"
)

type CalendarRepo struct {
	db *sql.DB
}

func NewCalendarRepo(db *sql.DB) *CalendarRepo {
	return &CalendarRepo{db: db}
}

func (repo *CalendarRepo) GetAllCalendars() ([]models.Calendar, error) {
	rows, err := repo.db.Query("SELECT id, name, ics_url, code, etag, last_modified, content_hash, last_checked_at FROM calendars ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []models.Calendar
	for rows.Next() {
		var calendar models.Calendar
		if err := rows.Scan(&calendar.ID, &calendar.Name, &calendar.ICS_url, &calendar.Code, &calendar.ETag, &calendar.LastModified, &calendar.ContentHash, &calendar.LastChecked); err != nil {
			return nil, err
		}
		calendars = append(calendars, calendar)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return calendars, nil
}

func (repo *CalendarRepo) GetSingleCalendar(code string) (*models.Calendar, error) {
	row := repo.db.QueryRow("SELECT id, name, ics_url, code, etag, last_modified, content_hash, last_checked_at FROM calendars WHERE code = $1", code)
	var calendar models.Calendar
	if err := row.Scan(&calendar.ID, &calendar.Name, &calendar.ICS_url, &calendar.Code, &calendar.ETag, &calendar.LastModified, &calendar.ContentHash, &calendar.LastChecked); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &calendar, nil
}

func (repo *CalendarRepo) UpdateCalendars(calendars []models.Calendar) error {
	tx, err := repo.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(`
		INSERT INTO calendars (code, name, ics_url, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (code)
		DO UPDATE SET
			name = EXCLUDED.name,
			ics_url = EXCLUDED.ics_url,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, calendar := range calendars {
		if calendar.Code == "" || calendar.Name == "" || calendar.ICS_url == "" {
			return fmt.Errorf("calendar has empty required field: %+v", calendar)
		}

		if _, err = stmt.Exec(calendar.Code, calendar.Name, calendar.ICS_url); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (repo *CalendarRepo) ListCalendarsForSync() ([]models.Calendar, error) {
	rows, err := repo.db.Query(`
		SELECT id, code, name, ics_url, etag, last_modified, content_hash, last_checked_at
		FROM calendars
		ORDER BY code, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []models.Calendar
	for rows.Next() {
		var calendar models.Calendar
		if err := rows.Scan(&calendar.ID, &calendar.Code, &calendar.Name, &calendar.ICS_url, &calendar.ETag, &calendar.LastModified, &calendar.ContentHash, &calendar.LastChecked); err != nil {
			return nil, err
		}
		calendars = append(calendars, calendar)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return calendars, nil
}

func (repo *CalendarRepo) UpdateCalendarSyncState(calendarID int, etag, lastModified, contentHash *string, checkedAt time.Time, hasChanged bool) error {
	if hasChanged {
		_, err := repo.db.Exec(`
			UPDATE calendars
			SET etag = $1,
				last_modified = $2,
				content_hash = $3,
				last_checked_at = $4,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $5
		`, etag, lastModified, contentHash, checkedAt, calendarID)
		return err
	}

	_, err := repo.db.Exec(`
		UPDATE calendars
		SET etag = $1,
			last_modified = $2,
			last_checked_at = $3
		WHERE id = $4
	`, etag, lastModified, checkedAt, calendarID)
	return err
}

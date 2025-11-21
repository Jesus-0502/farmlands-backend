package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"farmlands-backend/handlers"
	"farmlands-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// -----------------------------
// Router para logs
// -----------------------------
func buildLogsRouter(h *handlers.LogsHandler) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/log", h.HandleAddLog).Methods("POST")
	r.HandleFunc("/log", h.HandleSearchLog).Methods("GET")
	r.HandleFunc("/log/delete", h.HandleDeleteLog).Methods("POST")
	return r
}

//
// ──────────────────────────────────────────────
//   TESTS: HandleAddLog
// ──────────────────────────────────────────────
//

func TestHandleAddLog_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	body := models.Log{
		Event:     "UserLogin",
		Module:    "Auth",
		CreatedAt: "12-02-2025",
		Timestamp: "10:30 am",
		UserID:    1,
	}

	b, _ := json.Marshal(body)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO events (event, module, created_at, timestamp, fk_user)
		VALUES (?, ?, ?, ?, ?)
	`)).
		WithArgs(body.Event, body.Module, body.CreatedAt, body.Timestamp, body.UserID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleAddLog_InvalidJSON(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewBufferString(`{invalid_json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleAddLog_InvalidTimestamp(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	body := models.Log{
		Event:     "Test",
		Module:    "Test",
		CreatedAt: "12-02-2025",
		Timestamp: "25:99 pm", // ❌ inválido
		UserID:    1,
	}

	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleAddLog_InvalidDate(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	body := models.Log{
		Event:     "Test",
		Module:    "Test",
		CreatedAt: "99-99-9999", // ❌ inválido
		Timestamp: "10:30 am",
		UserID:    1,
	}

	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleAddLog_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	body := models.Log{
		Event:     "UserLogin",
		Module:    "Auth",
		CreatedAt: "12-02-2025",
		Timestamp: "10:30 am",
		UserID:    1,
	}

	b, _ := json.Marshal(body)

	mock.ExpectExec("INSERT INTO events").
		WithArgs(body.Event, body.Module, body.CreatedAt, body.Timestamp, body.UserID).
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodPost, "/log", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

//
// ──────────────────────────────────────────────
//   TESTS: HandleDeleteLog
// ──────────────────────────────────────────────
//

func TestHandleDeleteLog_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	body := models.LogID{ID: 1}
	b, _ := json.Marshal(body)

	mock.ExpectExec("DELETE FROM events WHERE id = ?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row deleted

	req := httptest.NewRequest(http.MethodPost, "/log/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleDeleteLog_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	body := models.LogID{ID: 10}
	b, _ := json.Marshal(body)

	mock.ExpectExec("DELETE FROM events WHERE id = ?").
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(0, 0)) // no rows

	req := httptest.NewRequest(http.MethodPost, "/log/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleDeleteLog_InvalidJSON(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/log/delete", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleDeleteLog_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	body := models.LogID{ID: 1}
	b, _ := json.Marshal(body)

	mock.ExpectExec("DELETE FROM events WHERE id = ?").
		WithArgs(1).
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodPost, "/log/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

//
// ──────────────────────────────────────────────
//   TESTS: HandleSearchLog
// ──────────────────────────────────────────────
//

func TestHandleSearchLog_All(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	rows := sqlmock.NewRows([]string{
		"id", "event", "module", "created_at", "username", "timestamp",
	}).AddRow(1, "Login", "Auth", "12-02-2025", "admin", "10:00 am")

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT e.id, e.event, e.module, e.created_at, u.username, e.timestamp FROM events e INNER JOIN users u ON e.fk_user = u.id",
	)).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleSearchLog_Query(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	query := "%admin%"

	rows := sqlmock.NewRows([]string{
		"id", "event", "module", "created_at", "username", "timestamp",
	}).AddRow(1, "Login", "Auth", "12-02-2025", "admin", "10:00 am")

	mock.ExpectQuery("SELECT e.id, e.event, e.module").
		WithArgs(query).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/log?q=admin", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleSearchLog_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewLogsHandler(db)
	router := buildLogsRouter(h)

	mock.ExpectQuery("SELECT e.id").
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/log?q=test", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

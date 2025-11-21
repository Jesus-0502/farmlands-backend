package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"farmlands-backend/handlers"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// -------------------------
// Router de pruebas
// -------------------------
func buildUnitsRouter(h *handlers.UnitsHandler) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/units", h.HandleAddUnit).Methods("POST")
	r.HandleFunc("/units", h.HandleSearchUnit).Methods("GET")
	r.HandleFunc("/units/measurements", h.HandleListMeasurements).Methods("GET")
	r.HandleFunc("/units/delete", h.HandleDeleteUnit).Methods("POST")
	r.HandleFunc("/units/edit", h.HandleEditUnits).Methods("POST")
	return r
}

//
// ---------------------------------------------------------
// TESTS HandleListMeasurements
// ---------------------------------------------------------
func TestHandleListMeasurements_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	rows := sqlmock.NewRows([]string{"id", "unit"}).
		AddRow(1, "KG").
		AddRow(2, "L")

	mock.ExpectQuery("SELECT id, unit FROM units").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/units/measurements", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

//
// ---------------------------------------------------------
// TESTS HandleAddUnit
// ---------------------------------------------------------
func TestHandleAddUnit_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	body := map[string]interface{}{
		"dimension": "10",
		"unit":      1,
	}
	b, _ := json.Marshal(body)

	// Mock existencia de unidad
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM units WHERE id = ?")).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	// Mock insert
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO measurements (dimension, fk_unit) VALUES (?, ?)")).
		WithArgs("10", 1).
		WillReturnResult(sqlmock.NewResult(5, 1))

	req := httptest.NewRequest(http.MethodPost, "/units", bytes.NewReader(b))
	rec := httptest.NewRecorder()
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleAddUnit_UnitNotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	body := map[string]interface{}{
		"dimension": "10",
		"unit":      999,
	}
	b, _ := json.Marshal(body)

	mock.ExpectQuery("SELECT 1 FROM units WHERE id = ?").
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/units", bytes.NewReader(b))
	rec := httptest.NewRecorder()
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

//
// ---------------------------------------------------------
// TESTS HandleDeleteUnit
// ---------------------------------------------------------
func TestHandleDeleteUnit_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	body := map[string]interface{}{"id": 3}
	b, _ := json.Marshal(body)

	mock.ExpectExec("DELETE FROM measurements WHERE id = ?").
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/units/delete", bytes.NewReader(b))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleDeleteUnit_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	body := map[string]interface{}{"id": 100}
	b, _ := json.Marshal(body)

	mock.ExpectExec("DELETE FROM measurements WHERE id = ?").
		WithArgs(100).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(http.MethodPost, "/units/delete", bytes.NewReader(b))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

//
// ---------------------------------------------------------
// TESTS HandleSearchUnit
// ---------------------------------------------------------
func TestHandleSearchUnit_All(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	rows := sqlmock.NewRows([]string{"id", "dimension", "unit"}).
		AddRow(1, "25", "KG")

	mock.ExpectQuery("SELECT m.id, m.dimension, u.unit FROM measurements m INNER JOIN units u ON m.fk_unit = u.id").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/units", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleSearchUnit_Query(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	rows := sqlmock.NewRows([]string{"id", "dimension", "unit"}).
		AddRow(1, "10", "L")

	mock.ExpectQuery("SELECT m.id, m.dimension, u.unit FROM measurements m INNER JOIN units u ON m.fk_unit = u.id WHERE").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/units?q=L", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

//
// ---------------------------------------------------------
// TESTS HandleEditUnits
// ---------------------------------------------------------
func TestHandleEditUnits_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	body := map[string]interface{}{
		"id":        1,
		"dimension": "50",
		"unit":      "KG",
	}
	b, _ := json.Marshal(body)

	// Buscar medición existente
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT m.id, m.dimension, m.fk_unit
		FROM measurements m WHERE id = ?`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "dimension", "fk_unit"}).
			AddRow(1, "25", 2))

	// Buscar unidad actual
	mock.ExpectQuery("SELECT unit FROM units WHERE id = ?").
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"unit"}).AddRow("L"))

	// Validar nuevo unit
	mock.ExpectQuery("SELECT id FROM units WHERE unit = ?").
		WithArgs("KG").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))

	// Update
	mock.ExpectExec("UPDATE measurements SET").
		WithArgs("50", int64(9), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/units/edit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleEditUnits_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	h := handlers.NewUnitsHandler(db)
	router := buildUnitsRouter(h)

	body := map[string]interface{}{
		"id": 999,
	}
	b, _ := json.Marshal(body)

	mock.ExpectQuery("SELECT m.id, m.dimension, m.fk_unit FROM measurements m WHERE id = ?").
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/units/edit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

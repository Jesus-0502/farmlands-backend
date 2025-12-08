package handlers

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

type successWrapper[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type errorWrapper struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func buildActionPlanRouter(h *handlers.ActionPlanHandler) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/action_plan", h.HandleNewActionPlan).Methods("POST")
	r.HandleFunc("/action_plan/delete", h.HandleDeleteActionPlan).Methods("POST")
	return r
}

func TestHandleNewActionPlan_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	// Body válido
	body := map[string]interface{}{
		"actividad":       "Siembra de maíz",
		"laborAgronomica": int64(2),
		"id_responsable":  int64(10),
		"idProject":       int64(100),
		"fecha_inicio":    "2025-12-01",
		"fecha_cierre":    "2025-12-15",
		"cantidadHoras":   12.5,
	}
	b, _ := json.Marshal(body)

	// Expect Exec insert con 7 valores
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO action_plans 
		(id_project, id_farm_task, action_description, fecha_inicio, fecha_cierre, cantidad_horas, id_responsable)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)).
		WithArgs(int64(100), int64(2), "Siembra de maíz", "2025-12-01", "2025-12-15", 12.5, int64(10)).
		WillReturnResult(sqlmock.NewResult(123, 1)) // LastInsertId = 123

	req := httptest.NewRequest(http.MethodPost, "/action_plan", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Deserializar considerando el wrapper de éxito
	var resp successWrapper[models.ActionPlan]
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)

	ap := resp.Data
	assert.Equal(t, int64(123), ap.ID)
	assert.Equal(t, "Siembra de maíz", ap.Actividad)
	assert.Equal(t, int64(2), ap.LaborAgronomica)
	assert.Equal(t, int64(10), ap.IDResponsable)
	assert.Equal(t, "2025-12-01", ap.Fecha_Inicio)
	assert.Equal(t, "2025-12-15", ap.Fecha_Cierre)
	assert.Equal(t, 12.5, ap.CantidadHoras)
	assert.Equal(t, float64(0), ap.Monto)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleNewActionPlan_InvalidJSON(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/action_plan", bytes.NewBuffer([]byte("bad json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "INVALID_JSON", errResp.Error.Code)
}

func TestHandleNewActionPlan_MissingRequiredFields(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	// Faltan campos obligatorios: actividad vacía y IDs en 0
	body := map[string]interface{}{
		"actividad":       "",
		"laborAgronomica": int64(0),
		"id_responsable":  int64(0),
		"idProject":       int64(0),
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/action_plan", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "INVALID_DATA", errResp.Error.Code)
}

func TestHandleNewActionPlan_InvalidFechaInicio(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	// fecha_inicio con formato inválido para el layout "2006-01-02"
	body := map[string]interface{}{
		"actividad":       "Riego",
		"laborAgronomica": int64(1),
		"id_responsable":  int64(2),
		"idProject":       int64(3),
		"fecha_inicio":    "01-12-2025", // inválido según time.Parse("2006-01-02")
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/action_plan", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "INVALID_DATE", errResp.Error.Code)
	assert.Contains(t, errResp.Error.Message, "fecha_inicio")
}

func TestHandleNewActionPlan_InvalidFechaCierre(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	// fecha_cierre con formato inválido para el layout "2006-01-02"
	body := map[string]interface{}{
		"actividad":       "Cosecha",
		"laborAgronomica": int64(1),
		"id_responsable":  int64(2),
		"idProject":       int64(3),
		"fecha_inicio":    "2025-12-01",
		"fecha_cierre":    "15-12-2025", // inválido
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/action_plan", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "INVALID_DATE", errResp.Error.Code)
	assert.Contains(t, errResp.Error.Message, "fecha_cierre")
}

func TestHandleNewActionPlan_DBErrorOnInsert(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	body := map[string]interface{}{
		"actividad":       "Aplicación de fertilizante",
		"laborAgronomica": int64(5),
		"id_responsable":  int64(20),
		"idProject":       int64(200),
		"fecha_inicio":    "2025-12-02",
		"fecha_cierre":    "2025-12-07",
		"cantidadHoras":   8.0,
	}
	b, _ := json.Marshal(body)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO action_plans 
		(id_project, id_farm_task, action_description, fecha_inicio, fecha_cierre, cantidad_horas, id_responsable)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)).
		WithArgs(int64(200), int64(5), "Aplicación de fertilizante", "2025-12-02", "2025-12-07", 8.0, int64(20)).
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodPost, "/action_plan", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "DB_ERROR", errResp.Error.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeleteActionPlan_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	body := models.ActionPlanID{ID: 123}
	b, _ := json.Marshal(body)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM action_plans WHERE id = ?")).
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 1)) // RowsAffected = 1

	req := httptest.NewRequest(http.MethodPost, "/action_plan/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp successWrapper[string]
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Data, "eliminado")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeleteActionPlan_InvalidJSON(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/action_plan/delete", bytes.NewBuffer([]byte("bad json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "INVALID_JSON", errResp.Error.Code)
}

func TestHandleDeleteActionPlan_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	body := models.ActionPlanID{ID: 999}
	b, _ := json.Marshal(body)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM action_plans WHERE id = ?")).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodPost, "/action_plan/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "DB_ERROR", errResp.Error.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeleteActionPlan_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	h := handlers.NewActionPlanHandler(db)
	router := buildActionPlanRouter(h)

	body := models.ActionPlanID{ID: 456}
	b, _ := json.Marshal(body)

	// Delete no afecta filas
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM action_plans WHERE id = ?")).
		WithArgs(int64(456)).
		WillReturnResult(sqlmock.NewResult(0, 0)) // RowsAffected = 0

	req := httptest.NewRequest(http.MethodPost, "/action_plan/delete", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp errorWrapper
	err := json.Unmarshal(rec.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.False(t, errResp.Success)
	assert.Equal(t, "NOT_FOUND", errResp.Error.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

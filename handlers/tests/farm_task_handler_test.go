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

func buildFarmTasksRouter(h *handlers.FarmTasksHandler) http.Handler {
    r := mux.NewRouter()
    r.HandleFunc("/farm_tasks/edit", h.HandleEditFarmTask).Methods("POST")
    r.HandleFunc("/farm_tasks/delete", h.HandleDeleteFarmTask).Methods("POST")
    r.HandleFunc("/farm_tasks", h.HandleAddFarmTask).Methods("POST")
    r.HandleFunc("/farm_tasks", h.HandleSearchFarmTask).Methods("GET")
    return r
}

func TestHandleAddFarmTask_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    body := models.AddFarmTask{Descripcion: "Nueva tarea"}
    b, _ := json.Marshal(body)

    mock.ExpectExec(regexp.QuoteMeta("INSERT INTO farm_tasks (descripcion) VALUES (?)")).
        WithArgs("Nueva tarea").
        WillReturnResult(sqlmock.NewResult(1, 1))

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleAddFarmTask_InvalidJSON(t *testing.T) {
    db, _, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks", bytes.NewBuffer([]byte("bad json")))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleAddFarmTask_MissingDescripcion(t *testing.T) {
    db, _, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    body := models.AddFarmTask{Descripcion: ""}
    b, _ := json.Marshal(body)

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleEditFarmTask_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    payload := map[string]interface{}{"id": 1, "descripcion": "Nuevo"}
    b, _ := json.Marshal(payload)

    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, descripcion FROM farm_tasks WHERE id = ?")).
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows([]string{"id", "descripcion"}).AddRow(1, "Viejo"))

    mock.ExpectExec(regexp.QuoteMeta("UPDATE farm_tasks SET descripcion = ? WHERE id = ?")).
        WithArgs("Nuevo", 1).
        WillReturnResult(sqlmock.NewResult(0, 1))

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks/edit", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleEditFarmTask_NoChanges(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    payload := map[string]interface{}{"id": 1, "descripcion": "Igual"}
    b, _ := json.Marshal(payload)

    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, descripcion FROM farm_tasks WHERE id = ?")).
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows([]string{"id", "descripcion"}).AddRow(1, "Igual"))

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks/edit", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleEditFarmTask_NotFound(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    payload := map[string]interface{}{"id": 10, "descripcion": "Nueva"}
    b, _ := json.Marshal(payload)

    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, descripcion FROM farm_tasks WHERE id = ?")).
        WithArgs(10).
        WillReturnError(sql.ErrNoRows)

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks/edit", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleDeleteFarmTask_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    body := models.FarmTaskID{ID: 3}
    b, _ := json.Marshal(body)

    mock.ExpectExec(regexp.QuoteMeta("DELETE FROM farm_tasks WHERE id = ?")).
        WithArgs(3).
        WillReturnResult(sqlmock.NewResult(0, 1))

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks/delete", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeleteFarmTask_NotFound(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewFarmTasksHandler(db)
    router := buildFarmTasksRouter(h)

    body := models.FarmTaskID{ID: 3}
    b, _ := json.Marshal(body)

    mock.ExpectExec(regexp.QuoteMeta("DELETE FROM farm_tasks WHERE id = ?")).
        WithArgs(3).
        WillReturnResult(sqlmock.NewResult(0, 0))

    req := httptest.NewRequest(http.MethodPost, "/farm_tasks/delete", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusNotFound, rec.Code)
}

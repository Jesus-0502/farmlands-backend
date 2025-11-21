package handlers_test

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "farmlands-backend/handlers"
    "farmlands-backend/models"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/gorilla/mux"
    "github.com/stretchr/testify/assert"
)

func buildRouter(h *handlers.ToolsHandler) http.Handler {
    r := mux.NewRouter()
	r.HandleFunc("/tools/edit", h.HandleEditTool).Methods("POST")
    r.HandleFunc("/tools/delete", h.HandleDeleteTool).Methods("POST")
    r.HandleFunc("/tools", h.HandleAddTool).Methods("POST")
    r.HandleFunc("/tools", h.HandleSearchTool).Methods("GET")

    return r
}

/*********************** LIST ***************************/

func TestHandleListTools_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    rows := sqlmock.NewRows([]string{"id", "descripcion"}).
        AddRow(1, "Pala").AddRow(2, "Tractor")

    mock.ExpectQuery("SELECT id, descripcion FROM tools").
        WillReturnRows(rows)

    req := httptest.NewRequest(http.MethodGet, "/tools", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleListTools_DBError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    mock.ExpectQuery("SELECT id, descripcion FROM tools").
        WillReturnError(sql.ErrConnDone)

    req := httptest.NewRequest(http.MethodGet, "/tools", nil)
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusInternalServerError, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

/*********************** ADD ***************************/

func TestHandleAddTool_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.AddTool{Descripcion: "Machete"}
    b, _ := json.Marshal(body)

    mock.ExpectExec("INSERT INTO tools").
        WithArgs("Machete").
        WillReturnResult(sqlmock.NewResult(1, 1))

    req := httptest.NewRequest(http.MethodPost, "/tools", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleAddTool_InvalidJSON(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    req := httptest.NewRequest(http.MethodPost, "/tools", bytes.NewBufferString("{bad json}"))
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleAddTool_MissingDescripcion(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.AddTool{}
    b, _ := json.Marshal(body)

    req := httptest.NewRequest(http.MethodPost, "/tools", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusBadRequest, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleAddTool_DBError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.AddTool{Descripcion: "Hacha"}
    b, _ := json.Marshal(body)

    mock.ExpectExec("INSERT INTO tools").
        WithArgs("Hacha").
        WillReturnError(sql.ErrConnDone)

    req := httptest.NewRequest(http.MethodPost, "/tools", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusInternalServerError, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

/*********************** EDIT ***************************/

func TestHandleEditTool_Success(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()


	h := handlers.NewToolsHandler(db)
	router := buildRouter(h)


	payload := map[string]interface{}{
	"id": 1,
	"descripcion": "Nuevo",
	}
	b, _ := json.Marshal(payload)


	mock.ExpectQuery("SELECT id, descripcion FROM tools WHERE id = ?").
	WithArgs(1).
	WillReturnRows(sqlmock.NewRows([]string{"id", "descripcion"}).AddRow(1, "Viejo"))


	mock.ExpectExec(`UPDATE tools SET descripcion = \? WHERE id = \?`).
		WithArgs("Nuevo", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/tools/edit", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)


	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleEditTool_NoChanges(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.Tool{ID: 1, Descripcion: "Igual"}
    b, _ := json.Marshal(body)

    mock.ExpectQuery("SELECT id, descripcion FROM tools WHERE id = ?").
        WithArgs(1).
        WillReturnRows(sqlmock.NewRows([]string{"id", "descripcion"}).AddRow(1, "Igual"))

    req := httptest.NewRequest(http.MethodPost, "/tools/edit", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleEditTool_NotFound(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.Tool{ID: 10, Descripcion: "X"}
    b, _ := json.Marshal(body)

    mock.ExpectQuery("SELECT id, descripcion FROM tools WHERE id = ?").
        WithArgs(10).
        WillReturnError(sql.ErrNoRows)

    req := httptest.NewRequest(http.MethodPost, "/tools/edit", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusNotFound, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleEditTool_DBErrorSelect(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.Tool{ID: 1, Descripcion: "X"}
    b, _ := json.Marshal(body)

    mock.ExpectQuery("SELECT id, descripcion FROM tools WHERE id = ?").
        WithArgs(1).
        WillReturnError(sql.ErrConnDone)

    req := httptest.NewRequest(http.MethodPost, "/tools/edit", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusInternalServerError, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

/*********************** DELETE ***************************/

func TestHandleDeleteTool_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.ToolID{ID: 3}
    b, _ := json.Marshal(body)

    mock.ExpectExec("DELETE FROM tools WHERE id = ?").
        WithArgs(3).
        WillReturnResult(sqlmock.NewResult(0, 1))

    req := httptest.NewRequest(http.MethodPost, "/tools/delete", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleDeleteTool_NotFound(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    h := handlers.NewToolsHandler(db)
    router := buildRouter(h)

    body := models.ToolID{ID: 3}
    b, _ := json.Marshal(body)

    mock.ExpectExec("DELETE FROM tools WHERE id = ?").
        WithArgs(3).
        WillReturnResult(sqlmock.NewResult(0, 0))

    req := httptest.NewRequest(http.MethodPost, "/tools/delete", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusNotFound, rec.Code)
    assert.NoError(t, mock.ExpectationsWereMet())
}



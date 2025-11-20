package handlers

import (
	"database/sql"
	"encoding/json"
	"farmlands-backend/models"
	"farmlands-backend/utils"
	"fmt"
	"log"
	"net/http"
	"regexp"
)

type LogsHandler struct {
	DB *sql.DB
}

func NewLogsHandler(db *sql.DB) *LogsHandler {
	return &LogsHandler{DB: db}
}

func (h *LogsHandler) HandleAddLog(w http.ResponseWriter, r *http.Request) {
	var input models.Log
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Println("Error decodificando JSON:", err)
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}
	fmt.Println("Log recibido:", input)

	// Validación del timestamp
	regex := regexp.MustCompile(`^(0[1-9]|1[0-2]):([0-5][0-9]) (am|pm)$`)
	if !regex.MatchString(input.Timestamp) {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_TIMESTAMP", "Formato de hora inválido. Use HH:MM am/pm")
		return
	}

	// Validar fecha con formato DD-MM-YYYY
	reDate := regexp.MustCompile(`^(0[1-9]|[12][0-9]|3[01])-(0[1-9]|1[0-2])-[0-9]{4}$`)
	if !reDate.MatchString(input.CreatedAt) {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_DATE", "La fecha debe tener el formato DD-MM-YYYY")
		return
	}

	stmt := `
		INSERT INTO events (event, module, created_at, timestamp, fk_user)
		VALUES (?, ?, ?, ?, ?)
	`

	res, err := h.DB.Exec(stmt, input.Event, input.Module, input.CreatedAt, input.Timestamp, input.UserID)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error logueando evento")
		return
	}

	id, _ := res.LastInsertId()

	log := struct {
		ID        int64
		Event     string
		Module    string
		CreatedAt string
		Timestamp string
		UserID    int64
	}{
		ID:        id,
		Event:     input.Event,
		Module:    input.Module,
		CreatedAt: input.CreatedAt,
		Timestamp: input.Timestamp,
		UserID:    input.UserID,
	}

	utils.SendJSONSuccess(w, log)
}

func (h *LogsHandler) HandleDeleteLog(w http.ResponseWriter, r *http.Request) {
	var input models.LogID

	// Decodificar JSON de entrada
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	res, err := h.DB.Exec("DELETE FROM events WHERE id = ?", input.ID)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error eliminando log")
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		utils.SendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Log no encontrado")
		return
	}

	utils.SendJSONSuccess(w, "Log eliminado correctamente")
}

func (h *LogsHandler) HandleSearchLog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q") // texto buscado

	// Si no hay query, devolvemos todos los suplementos
	var rows *sql.Rows
	var err error
	if query == "" {
		rows, err = h.DB.Query("SELECT e.id, e.event, e.module, e.created_at, u.username, e.timestamp FROM events e INNER JOIN users u ON e.fk_user = u.id")
	} else {
		rows, err = h.DB.Query(
			"SELECT e.id, e.event, e.module, e.created_at, u.username, e.timestamp FROM events e INNER JOIN users u ON e.fk_user = u.id WHERE UPPER(u.username) LIKE UPPER($1) OR UPPER(event) LIKE UPPER($1) OR UPPER(module) LIKE UPPER($1) OR timestamp LIKE $1",
			"%"+query+"%",
		)
	}

	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error en la búsqueda")
		return
	}
	defer rows.Close()

	type Log struct {
		ID        int64  `json:"id"`
		Event     string `json:"event"`
		Module    string `json:"module"`
		CreatedAt string `json:"created_at"`
		Timestamp string `json:"timestamp"`
		By        string `json:"by"`
	}

	var logs []Log
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.Event, &l.Module, &l.CreatedAt, &l.By, &l.Timestamp); err != nil {
			utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error leyendo resultados")
			return
		}
		logs = append(logs, l)
	}

	if logs == nil {
		logs = []Log{}
	}

	utils.SendJSONSuccess(w, logs)
}

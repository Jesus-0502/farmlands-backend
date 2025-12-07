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
	"strings"
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
func flipDateFormat(in string) string {
	parts := strings.Split(in, "-")
	if len(parts) != 3 {
		return in
	}
	return parts[2] + "-" + parts[1] + "-" + parts[0]
}
func getMonth(in string) string {
	parts := strings.Split(in, "-")
	if len(parts) != 3 {
		return ""
	}
	return parts[1]
}

func getYear(in string) string {
	parts := strings.Split(in, "-")
	if len(parts) != 3 {
		return ""
	}
	return parts[0]
}

func (h *LogsHandler) HandleSearchLog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	filter := r.URL.Query().Get("f")
	content := r.URL.Query().Get("c")

	baseQuery := `
		SELECT e.id, e.event, e.module, e.created_at, u.username, e.timestamp
		FROM events e
		INNER JOIN users u ON e.fk_user = u.id
	`

	extraFilter := ""
	// convierte "YYYY-MM-DD" a "DD-MM-YYYY"

	args := []interface{}{}

	// --------------------------------------------------------
	// Aplicar filtro SOLO si f y c existen ambos
	// --------------------------------------------------------

	if filter != "" && content != "" {

		switch filter {

		case "day":
			converted := flipDateFormat(content)
			log.Println("FECHA ANTES:", content)
			log.Println("FECHA DESPUES:", converted)
			extraFilter = " WHERE e.created_at LIKE $1"
			args = append(args, converted)

		case "month":
			month := content // "12"
			extraFilter = " WHERE SUBSTRING(e.created_at, 4, 2) LIKE $1"
			args = append(args, month)

		case "quarter":
			var q string = strings.ToUpper(content)

			// Caso 1: El usuario envía Q1, Q2, Q3, Q4
			if q == "Q1" {
				extraFilter = " WHERE SUBSTRING(e.created_at, 4, 2) IN ('01','02','03')"
			} else if q == "Q2" {
				extraFilter = " WHERE SUBSTRING(e.created_at, 4, 2) IN ('04','05','06')"
			} else if q == "Q3" {
				extraFilter = " WHERE SUBSTRING(e.created_at, 4, 2) IN ('07','08','09')"
			} else if q == "Q4" {
				extraFilter = " WHERE SUBSTRING(e.created_at, 4, 2) IN ('10','11','12')"
			}

		case "year":
			c := strings.TrimSpace(content)

			// Si viene solo el año: "2025"
			if len(c) == 4 {
				extraFilter = " WHERE SUBSTRING(e.created_at, 7, 4) = $1"
				args = append(args, c)
				break
			}

			// Si viene como fecha "2025-12-06"
			year := getYear(c)
			if year != "" {
				extraFilter = " WHERE SUBSTRING(e.created_at, 7, 4) = $1"
				args = append(args, year)
			}

		case "period":
			parts := strings.Split(content, ",")
			if len(parts) == 2 {
				start := flipDateFormat(strings.TrimSpace(parts[0])) // ahora retorna YYYY-MM-DD
				end := flipDateFormat(strings.TrimSpace(parts[1]))

				extraFilter = `
					WHERE 
						SUBSTR(e.created_at, 7, 4) || '-' || 
						SUBSTR(e.created_at, 4, 2) || '-' || 
						SUBSTR(e.created_at, 1, 2)
					BETWEEN $1 AND $2
				`
				args = append(args, start, end)
			}
		}
	}
	// --------------------------------------------------------
	// Agregar búsqueda textual si existe query q
	// --------------------------------------------------------

	var rows *sql.Rows
	var err error

	if query != "" {
		search := "%" + query + "%"

		if extraFilter == "" {
			extraFilter = " WHERE "
		} else {
			extraFilter += " AND "
		}

		fullQuery := baseQuery + extraFilter + `
			(
				UPPER(u.username) LIKE UPPER($1) 
				OR UPPER(e.event) LIKE UPPER($1) 
				OR UPPER(e.module) LIKE UPPER($1) 
				OR e.created_at LIKE $1
			)
		`

		rows, err = h.DB.Query(fullQuery, append(args, search)...)

	} else {
		// Sin texto buscado
		fullQuery := baseQuery + extraFilter
		rows, err = h.DB.Query(fullQuery, args...)
	}

	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error en la búsqueda")
		log.Println("Error decodificando JSON:", err)
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

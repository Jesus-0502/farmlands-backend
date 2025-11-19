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

func NormalizeNumberString(s string) (string, error) {
	// 1. Quitar espacios intermedios
	value := strings.ReplaceAll(s, " ", "")

	// 2. Validar formato general: entero | decimal | fracción
	re := regexp.MustCompile(`^(\d+|\d+\.\d+|\d+/\d+)$`)
	if !re.MatchString(value) {
		return "", fmt.Errorf("formato inválido: debe ser entero, decimal o fracción A/B")
	}

	// 3. Validación especial de fracciones
	if strings.Contains(value, "/") {
		parts := strings.Split(value, "/")
		num := parts[0]
		den := parts[1]
		// Normalizar ceros iniciales
		numNormalized := strings.TrimLeft(num, "0")
		if numNormalized == "" {
			numNormalized = "0"
			return "0", nil
		}
		// Validar denominador distinto de 0
		if den == "" || regexp.MustCompile(`^0+$`).MatchString(den) {
			return "", fmt.Errorf("el denominador de la fracción no puede ser 0")
		}
		denNormalized := strings.TrimLeft(den, "0")
		if denNormalized == "" {
			denNormalized = "0"
			return "", fmt.Errorf("el denominador de la fracción no puede ser 0")
		}

		value = numNormalized + "/" + denNormalized
		// Fracción es válida, return tal cual (no se normaliza)
		return value, nil
	}

	// 4. Si es entero, normalizar ceros iniciales
	if regexp.MustCompile(`^\d+$`).MatchString(value) {
		normalized := strings.TrimLeft(value, "0")
		if normalized == "" {
			normalized = "0"
		}
		return normalized, nil
	}

	// 5. Si es decimal, devolverlo tal cual (no eliminar ceros)
	return value, nil
}

type UnitsHandler struct {
	DB *sql.DB
}

func NewUnitsHandler(db *sql.DB) *UnitsHandler {
	return &UnitsHandler{DB: db}
}

func (h *UnitsHandler) HandleAddUnit(w http.ResponseWriter, r *http.Request) {
	var input models.Units
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Println("Error decodificando JSON:", err)
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	if input.Dimension == "" || input.Unit == 0 {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_INPUT", "La dimension y la unidad son obligatorias")
		return
	}

	var exists int
	err := h.DB.QueryRow("SELECT 1 FROM units WHERE id = ?", input.Unit).Scan(&exists)
	if err == sql.ErrNoRows {
		utils.SendJSONError(w, http.StatusBadRequest, "UNIT_NOT_FOUND", "La unidad especificada no existe")
		return
	}
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error verificando la unidad")
		return
	}

	normalized, err := NormalizeNumberString(input.Dimension)
	if err != nil {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_NUMBER", err.Error())
		return
	}

	if normalized == "0" {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_NUMBER", "La dimensión no puede ser cero")
		return
	}
	// Si todo ok, usar normalized
	input.Dimension = normalized

	stmt := `
		INSERT INTO measurements (dimension, fk_unit)
		VALUES (?, ?)
	`

	res, err := h.DB.Exec(stmt, input.Dimension, input.Unit)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error agregando nueva unidad de medida")
		return
	}

	id, _ := res.LastInsertId()

	measurement := struct {
		ID        int64
		Dimension string
		Unit      int64
	}{
		ID:        id,
		Dimension: input.Dimension,
		Unit:      input.Unit,
	}

	utils.SendJSONSuccess(w, measurement)
}

func (h *UnitsHandler) HandleDeleteUnit(w http.ResponseWriter, r *http.Request) {
	var input models.ToolID

	// Decodificar JSON de entrada
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	res, err := h.DB.Exec("DELETE FROM measurements WHERE id = ?", input.ID)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error eliminando unidad de medida")
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		utils.SendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Unidad de medida no encontrada")
		return
	}

	utils.SendJSONSuccess(w, "Unidad de medida eliminada correctamente")
}

func (h *UnitsHandler) HandleSearchUnit(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q") // texto buscado

	// Si no hay query, devolvemos todos los suplementos
	var rows *sql.Rows
	var err error
	if query == "" {
		rows, err = h.DB.Query("SELECT m.id, m.dimension, u.unit FROM measurements m INNER JOIN units u ON m.fk_unit = u.id")
	} else {
		rows, err = h.DB.Query(
			"SELECT m.id, m.dimension, u.unit FROM measurements m INNER JOIN units u ON m.fk_unit = u.id WHERE UPPER(m.dimension) LIKE UPPER($1) OR UPPER(u.unit) LIKE UPPER($1)",
			"%"+query+"%",
		)
	}

	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error en la búsqueda")
		return
	}
	defer rows.Close()

	type Measurements struct {
		ID        int64  `json:"id"`
		Dimension string `json:"dimension"`
		Unit      string `json:"unit"`
	}

	var measurements []Measurements
	for rows.Next() {
		var m Measurements
		if err := rows.Scan(&m.ID, &m.Dimension, &m.Unit); err != nil {
			utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error leyendo resultados")
			return
		}
		measurements = append(measurements, m)
	}

	utils.SendJSONSuccess(w, measurements)
}

func (h *UnitsHandler) HandleEditUnits(w http.ResponseWriter, r *http.Request) {
	type Measurements struct {
		ID        int64  `json:"id"`
		Dimension string `json:"dimension"`
		Unit      string `json:"unit"` // este es el string que el usuario envía
	}

	var input Measurements

	// Decodificar JSON de entrada
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	// Validar ID obligatorio
	if input.ID == 0 {
		utils.SendJSONError(w, http.StatusBadRequest, "MISSING_ID", "El ID de la unidad de medida es obligatorio")
		return
	}

	// Buscar la medición actual
	var dbMeasurement struct {
		ID        int64
		Dimension string
		FKUnit    int64
	}

	err := h.DB.QueryRow(`
		SELECT m.id, m.dimension, m.fk_unit
		FROM measurements m WHERE id = ?`,
		input.ID,
	).Scan(&dbMeasurement.ID, &dbMeasurement.Dimension, &dbMeasurement.FKUnit)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.SendJSONError(w, http.StatusNotFound, "MEASUREMENT_NOT_FOUND", "Unidad de medida no encontrada")
			return
		}
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error consultando unidad de medida")
		return
	}

	// Para responder, necesitamos devolver el unit como string real
	var currentUnitString string
	err = h.DB.QueryRow(`SELECT unit FROM units WHERE id = ?`, dbMeasurement.FKUnit).
		Scan(&currentUnitString)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error consultando unidad actual")
		return
	}

	// Construcción dinámica
	updateFields := []string{}
	args := []interface{}{}

	// --- DIMENSION ---
	if input.Dimension != "" && input.Dimension != dbMeasurement.Dimension {
		updateFields = append(updateFields, "dimension = ?")
		args = append(args, input.Dimension)
		dbMeasurement.Dimension = input.Dimension
	}

	// --- UNIT ---
	if input.Unit != "" && input.Unit != currentUnitString {

		// 1. Validar que exista el unit del usuario
		var newUnitID int64
		err := h.DB.QueryRow(`
			SELECT id FROM units WHERE unit = ?
		`, input.Unit).Scan(&newUnitID)

		if err == sql.ErrNoRows {
			utils.SendJSONError(w, http.StatusBadRequest, "UNIT_NOT_FOUND", "La unidad especificada no existe")
			return
		}
		if err != nil {
			utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error consultando unidad")
			return
		}

		// 2. Agregar al UPDATE el ID encontrado
		updateFields = append(updateFields, "fk_unit = ?")
		args = append(args, newUnitID)
		dbMeasurement.FKUnit = newUnitID
		currentUnitString = input.Unit
	}

	// Si no hay cambios:
	if len(updateFields) == 0 {
		utils.SendJSONSuccess(w, map[string]string{"message": "No se realizaron cambios"})
		return
	}

	// Construcción final del UPDATE
	query := fmt.Sprintf("UPDATE measurements SET %s WHERE id = ?", strings.Join(updateFields, ", "))
	args = append(args, input.ID)

	_, err = h.DB.Exec(query, args...)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_UPDATE_ERROR", "Error actualizando unidad de medida")
		return
	}

	// Preparar respuesta final
	response := Measurements{
		ID:        dbMeasurement.ID,
		Dimension: dbMeasurement.Dimension,
		Unit:      currentUnitString,
	}

	utils.SendJSONSuccess(w, response)
}

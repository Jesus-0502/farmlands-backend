package handlers

import (
	"database/sql"
	"encoding/json"
	"farmlands-backend/models"
	"farmlands-backend/utils"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ActionPlanHandler struct {
	DB *sql.DB
}

func NewActionPlanHandler(db *sql.DB) *ActionPlanHandler {
	return &ActionPlanHandler{DB: db}
}

func (h *ActionPlanHandler) HandleNewActionPlan(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Actividad       string  `json:"actividad"`
		LaborAgronomica int64   `json:"laborAgronomica"`
		IDResponsable   int64   `json:"id_responsable"`
		IDProject       int64   `json:"idProject"`
		FechaInicio     string  `json:"fecha_inicio"`
		FechaCierre     string  `json:"fecha_cierre"`
		CantidadHoras   float64 `json:"cantidadHoras"`
	}

	// ---------- Decodificar JSON ----------
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Println("Error decodificando JSON:", err)
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	// ---------- Validación de campos obligatorios ----------
	if input.Actividad == "" || input.LaborAgronomica == 0 || input.IDResponsable == 0 || input.IDProject == 0 {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_DATA", "Campos obligatorios faltantes")
		return
	}

	// ---------- Insert SQL (sin monto) ----------
	stmt := `
		INSERT INTO action_plans 
		(id_project, id_farm_task, action_description, fecha_inicio, fecha_cierre, cantidad_horas, id_responsable)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	// Dentro del handler después de decodificar el JSON:
	const layout = "2006-01-02"

	if input.FechaInicio != "" {
		if _, err := time.Parse(layout, input.FechaInicio); err != nil {
			utils.SendJSONError(w, http.StatusBadRequest, "INVALID_DATE", "fecha_inicio debe tener formato DD-MM-AAAA")
			return
		}
	}

	if input.FechaCierre != "" {
		if _, err := time.Parse(layout, input.FechaCierre); err != nil {
			utils.SendJSONError(w, http.StatusBadRequest, "INVALID_DATE", "fecha_cierre debe tener formato DD-MM-AAAA")
			return
		}
	}
	res, err := h.DB.Exec(stmt,
		input.IDProject,
		input.LaborAgronomica,
		input.Actividad,
		input.FechaInicio,
		input.FechaCierre,
		input.CantidadHoras,
		input.IDResponsable,
	)

	if err != nil {
		log.Println("Error al insertar plan de acción:", err)
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error agregando plan de acción")
		return
	}

	idEntry, _ := res.LastInsertId()

	actionPlan := models.ActionPlan{
		ID:              idEntry,
		Actividad:       input.Actividad,
		LaborAgronomica: input.LaborAgronomica,
		IDResponsable:   input.IDResponsable,
		Fecha_Inicio:    input.FechaInicio,
		Fecha_Cierre:    input.FechaCierre,
		CantidadHoras:   input.CantidadHoras,
		Monto:           0, // Monto aún no calculado
	}

	utils.SendJSONSuccess(w, actionPlan)
}

func (h *ActionPlanHandler) HandleDeleteActionPlan(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID int64 `json:"id"`
	}

	// Decodificar JSON
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	// Ejecutar Delete
	res, err := h.DB.Exec("DELETE FROM action_plans WHERE id = ?", input.ID)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error eliminando plan de acción")
		return
	}

	// Verificar si realmente eliminó algo
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		utils.SendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Plan de acción no encontrado")
		return
	}

	utils.SendJSONSuccess(w, "Plan de acción eliminado correctamente")
}

func (h *ActionPlanHandler) HandleSearchActionPlan(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	idStr := r.URL.Query().Get("i")
	tStr := r.URL.Query().Get("t")

	// id_project obligatorio
	if idStr == "" {
		utils.SendJSONError(w, http.StatusBadRequest, "MISSING_ID", "El parámetro 'i' (id del proyecto) es obligatorio")
		return
	}

	idProject, errConv := strconv.ParseInt(idStr, 10, 64)
	if errConv != nil {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_ID", "El parámetro 'i' debe ser numérico")
		return
	}

	// parámetro t (0,1,2)
	t := 0
	if tStr != "" {
		t, errConv = strconv.Atoi(tStr)
		if errConv != nil || t < 0 || t > 2 {
			utils.SendJSONError(w, http.StatusBadRequest, "INVALID_T", "El parámetro 't' debe ser 0, 1 o 2")
			return
		}
	}

	// SELECT dinámico según t
	var baseSelect string
	switch t {
	case 0:
		baseSelect = `
			SELECT id, action_description, id_farm_task, id_responsable,
			fecha_inicio, fecha_cierre, cantidad_horas, monto_total
			FROM action_plans
			WHERE id_project = ?
		`
	case 1:
		baseSelect = `
			SELECT id, action_description, id_farm_task, cantidad_horas,
			tiempo_recurso_humano, id_responsable, costo_recurso_humano, monto_recurso_humano
			FROM action_plans
			WHERE id_project = ?
		`
	case 2:
		baseSelect = `
			SELECT id, action_description, id_farm_task, cantidad_horas,
			categoria_insumo_material, descripcion_insumo_material, cantidad_insumo_material,
			id_medida, costo_insumo_material, id_responsable, monto_insumo_material
			FROM action_plans
			WHERE id_project = ?
		`
	}

	// Ejecutar query
	var rows *sql.Rows
	var err error

	if query == "" {
		rows, err = h.DB.Query(baseSelect, idProject)
	} else {
		search := "%" + query + "%"
		rows, err = h.DB.Query(baseSelect+`
			AND (
				UPPER(CAST(id AS TEXT)) LIKE UPPER(?)
				OR UPPER(action_description) LIKE UPPER(?)
				OR UPPER(CAST(id_farm_task AS TEXT)) LIKE UPPER(?)
			)
		`, idProject, search, search, search)
	}

	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error en la búsqueda")
		return
	}
	defer rows.Close()

	// generar JSON dinámico
	results := []map[string]interface{}{}
	cols, _ := rows.Columns()

	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range cols {
			ptrs[i] = &values[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error leyendo resultados")
			return
		}

		item := make(map[string]interface{})
		for i, col := range cols {
			v := values[i]

			// Si es []byte → convertir a string
			if b, ok := v.([]byte); ok {
				item[col] = string(b)
				continue
			}

			// NULL → reemplazar por defaults
			if v == nil {
				switch col {
				case "id", "id_farm_task", "id_responsable", "cantidad_insumo_material", "id_medida":
					item[col] = int64(0)
				case "cantidad_horas", "tiempo_recurso_humano", "monto_recurso_humano",
					"costo_recurso_humano", "monto_insumo_material",
					"costo_insumo_material", "monto_total":
					item[col] = float64(0)
				default:
					item[col] = "" // strings
				}
				continue
			}

			item[col] = v
		}

		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error iterando resultados")
		return
	}

	utils.SendJSONSuccess(w, results)
}

func (h *ActionPlanHandler) HandleEditActionPlan(w http.ResponseWriter, r *http.Request) {
	var input models.ActionPlanEdit

	// 1. Decodificación del JSON (usando el struct original con primitivos)
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.SendJSONError(w, http.StatusBadRequest, "INVALID_JSON", "JSON inválido")
		return
	}

	if input.ID == 0 {
		utils.SendJSONError(w, http.StatusBadRequest, "MISSING_ID", "El ID del plan de acción es obligatorio")
		return
	}

	// 2. Validación de fechas (igual que antes)
	const layout = "2006-01-02"
	if input.Fecha_Inicio != "" {
		if _, err := time.Parse(layout, input.Fecha_Inicio); err != nil {
			utils.SendJSONError(w, http.StatusBadRequest, "INVALID_DATE", "fecha_inicio debe tener formato DD-MM-AAAA")
			return
		}
	}
	if input.Fecha_Cierre != "" {
		if _, err := time.Parse(layout, input.Fecha_Cierre); err != nil {
			utils.SendJSONError(w, http.StatusBadRequest, "INVALID_DATE", "fecha_cierre debe tener formato DD-MM-AAAA")
			return
		}
	}

	// 3. Construcción de query dinámica (igual que antes)
	query := "UPDATE action_plans SET "
	params := []interface{}{}
	changes := 0

	fields := []struct {
		value interface{}
		sql   string
	}{
		{input.ActionDescription, "action_description = ?"},
		{input.IDFarmTask, "id_farm_task = ?"},
		{input.IDResponsable, "id_responsable = ?"},
		{input.Fecha_Inicio, "fecha_inicio = ?"},
		{input.Fecha_Cierre, "fecha_cierre = ?"},
		{input.CantidadHoras, "cantidad_horas = ?"},
		{input.TiempoRecursoHumano, "tiempo_recurso_humano = ?"},
		{input.MontoRecursoHumano, "monto_recurso_humano = ?"},
		{input.CostoRecursoHumano, "costo_recurso_humano = ?"},
		{input.CategoriaInsumoMaterial, "categoria_insumo_material = ?"},
		{input.DescripcionInsumoMaterial, "descripcion_insumo_material = ?"},
		// NOTA: Asume que has corregido el nombre de la columna en la DB o aquí:
		{input.CantidadInsumoMaterial, "cantidad_insumo_material = ?"},
		{input.IDMedida, "id_medida = ?"},
		{input.MontoInsumoMaterial, "monto_insumo_material = ?"},
		{input.CostoInsumoMaterial, "costo_insumo_material = ?"},
		{input.MontoTotal, "monto_total = ?"},
	}

	for _, f := range fields {
		switch v := f.value.(type) {
		case string:
			if v != "" {
				query += f.sql + ", "
				params = append(params, v)
				changes++
			}
		case float64:
			if v != 0 {
				query += f.sql + ", "
				params = append(params, v)
				changes++
			}
		case int64:
			if v != 0 {
				query += f.sql + ", "
				params = append(params, v)
				changes++
			}
		}
	}

	if changes == 0 {
		utils.SendJSONError(w, http.StatusBadRequest, "NO_FIELDS", "No se enviaron campos válidos para modificar")
		return
	}

	// 4. Ejecución del UPDATE
	query = query[:len(query)-2] + " WHERE id = ?"
	params = append(params, input.ID)

	res, err := h.DB.Exec(query, params...)
	if err != nil {
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error actualizando plan de acción")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		utils.SendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Registro no encontrado")
		return
	}

	// =================================================================
	// 5. SOLUCIÓN AL ERROR DE SCAN (Usando struct anónimo temporal)
	// =================================================================

	// Definición de una estructura anónima TEMPORAL para escanear.
	// TODOS los campos que pueden ser NULL en la DB deben ser sql.Null*.
	var tempResult struct {
		ID                        int64
		IDProject                 sql.NullInt64
		IDFarmTask                sql.NullInt64
		ActionDescription         sql.NullString
		Fecha_Inicio              sql.NullString
		Fecha_Cierre              sql.NullString
		CantidadHoras             sql.NullFloat64
		IDResponsable             sql.NullInt64
		TiempoRecursoHumano       sql.NullFloat64 // ¡Este era el que fallaba!
		MontoRecursoHumano        sql.NullFloat64
		CostoRecursoHumano        sql.NullFloat64
		CategoriaInsumoMaterial   sql.NullString
		DescripcionInsumoMaterial sql.NullString
		CantidadInsumoMaterial    sql.NullInt64
		IDMedida                  sql.NullInt64
		MontoInsumoMaterial       sql.NullFloat64
		CostoInsumoMaterial       sql.NullFloat64
		MontoTotal                sql.NullFloat64
	}

	// Ejecución del SELECT (debe coincidir con los 18 campos anteriores)
	row := h.DB.QueryRow(`
        SELECT
            id,
            id_project,
            id_farm_task,
            action_description,
            fecha_inicio,
            fecha_cierre,
            cantidad_horas,
            id_responsable,
            tiempo_recurso_humano,
            monto_recurso_humano,
            costo_recurso_humano,
            categoria_insumo_material,
            descripcion_insumo_material,
            cantidad_insumo_material,
            id_medida,
            monto_insumo_material,
            costo_insumo_material,
            monto_total
        FROM action_plans
        WHERE id = ?
    `, input.ID)

	// El Scan ahora usa las referencias de la estructura temporal (sql.Null*)
	err = row.Scan(
		&tempResult.ID,
		&tempResult.IDProject,
		&tempResult.IDFarmTask,
		&tempResult.ActionDescription,
		&tempResult.Fecha_Inicio,
		&tempResult.Fecha_Cierre,
		&tempResult.CantidadHoras,
		&tempResult.IDResponsable,
		&tempResult.TiempoRecursoHumano,
		&tempResult.MontoRecursoHumano,
		&tempResult.CostoRecursoHumano,
		&tempResult.CategoriaInsumoMaterial,
		&tempResult.DescripcionInsumoMaterial,
		&tempResult.CantidadInsumoMaterial,
		&tempResult.IDMedida,
		&tempResult.MontoInsumoMaterial,
		&tempResult.CostoInsumoMaterial,
		&tempResult.MontoTotal,
	)

	if err != nil {
		// Maneja errores de DB/Scan (incluyendo un NULL inesperado)
		utils.SendJSONError(w, http.StatusInternalServerError, "DB_ERROR", "Error recuperando registro actualizado")
		return
	}

	// 6. Mapeo de vuelta al Struct de Respuesta (ActionPlanEdit)
	var result models.ActionPlanEdit

	// Mapeo del ID
	result.ID = tempResult.ID

	// Mapeo condicional: Solo asigna si el valor no es NULL (Valid = true)
	if tempResult.IDProject.Valid {
		result.IDProject = tempResult.IDProject.Int64
	}
	if tempResult.IDFarmTask.Valid {
		result.IDFarmTask = tempResult.IDFarmTask.Int64
	}
	if tempResult.ActionDescription.Valid {
		result.ActionDescription = tempResult.ActionDescription.String
	}
	if tempResult.Fecha_Inicio.Valid {
		result.Fecha_Inicio = tempResult.Fecha_Inicio.String
	}
	if tempResult.Fecha_Cierre.Valid {
		result.Fecha_Cierre = tempResult.Fecha_Cierre.String
	}
	if tempResult.CantidadHoras.Valid {
		result.CantidadHoras = tempResult.CantidadHoras.Float64
	}
	if tempResult.IDResponsable.Valid {
		result.IDResponsable = tempResult.IDResponsable.Int64
	}
	if tempResult.TiempoRecursoHumano.Valid {
		result.TiempoRecursoHumano = tempResult.TiempoRecursoHumano.Float64
	}
	if tempResult.MontoRecursoHumano.Valid {
		result.MontoRecursoHumano = tempResult.MontoRecursoHumano.Float64
	}
	if tempResult.CostoRecursoHumano.Valid {
		result.CostoRecursoHumano = tempResult.CostoRecursoHumano.Float64
	}
	if tempResult.CategoriaInsumoMaterial.Valid {
		result.CategoriaInsumoMaterial = tempResult.CategoriaInsumoMaterial.String
	}
	if tempResult.DescripcionInsumoMaterial.Valid {
		result.DescripcionInsumoMaterial = tempResult.DescripcionInsumoMaterial.String
	}
	if tempResult.CantidadInsumoMaterial.Valid {
		result.CantidadInsumoMaterial = tempResult.CantidadInsumoMaterial.Int64
	}
	if tempResult.IDMedida.Valid {
		result.IDMedida = tempResult.IDMedida.Int64
	}
	if tempResult.MontoInsumoMaterial.Valid {
		result.MontoInsumoMaterial = tempResult.MontoInsumoMaterial.Float64
	}
	if tempResult.CostoInsumoMaterial.Valid {
		result.CostoInsumoMaterial = tempResult.CostoInsumoMaterial.Float64
	}
	if tempResult.MontoTotal.Valid {
		result.MontoTotal = tempResult.MontoTotal.Float64
	}

	// 7. Respuesta de éxito
	utils.SendJSONSuccess(w, result)
}

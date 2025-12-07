package models

// Representación de planes de acción

type ActionPlanID struct {
	ID int64 `json:"id"`
}

type ActionPlan struct {
	ID              int64   `json:"id"`
	Actividad       string  `json:"actividad"`
	LaborAgronomica int64   `json:"laborAgronomica"`
	IDResponsable   int64   `json:"id_responsable"`
	Fecha_Inicio    string  `json:"fecha_inicio"`
	Fecha_Cierre    string  `json:"fecha_cierre"`
	CantidadHoras   float64 `json:"cantidadHoras"`
	Monto           float64 `json:"monto"`
}

type ActionPlanEdit struct {
	ID                        int64   `json:"id"`
	IDProject                 int64   `json:"id_project"`
	ActionDescription         string  `json:"action_description"`
	IDFarmTask                int64   `json:"id_farm_task"`
	IDResponsable             int64   `json:"id_responsable"`
	Fecha_Inicio              string  `json:"fecha_inicio"`
	Fecha_Cierre              string  `json:"fecha_cierre"`
	CantidadHoras             float64 `json:"cantidad_horas"`
	TiempoRecursoHumano       float64 `json:"tiempo_recurso_humano"`
	MontoRecursoHumano        float64 `json:"monto_recurso_humano"`
	CostoRecursoHumano        float64 `json:"costo_recurso_humano"`
	CategoriaInsumoMaterial   string  `json:"categoria_insumo_material"`
	DescripcionInsumoMaterial string  `json:"descripcion_insumo_material"`
	CantidadInsumoMaterial    int64   `json:"cantidad_insumo_material"`
	IDMedida                  int64   `json:"id_medida"`
	MontoInsumoMaterial       float64 `json:"monto_insumo_material"`
	CostoInsumoMaterial       float64 `json:"costo_insumo_material"`
	MontoTotal                float64 `json:"monto_total"`
}

package models

// Representación de unidades de medidas
type Units struct {
	Dimension string `json:"dimension"`
	Unit      int64  `json:"unit"`
}

type UnitID struct {
	ID int64 `json:"id"`
}

type MeasurementsUnits struct {
	ID   int64  `json:"id"`
	Unit string `json:"unit`
}

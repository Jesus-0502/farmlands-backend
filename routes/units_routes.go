package routes

import (
	"database/sql"
	"farmlands-backend/handlers"

	"github.com/gorilla/mux"
)

func RegisterUnitsRoutes(router *mux.Router, db *sql.DB) {
	handler := handlers.NewUnitsHandler(db)

	router.HandleFunc("/units", handler.HandleAddUnit).Methods("POST")
	router.HandleFunc("/units", handler.HandleSearchUnit).Methods("GET")
	router.HandleFunc("/units/delete", handler.HandleDeleteUnit).Methods("POST")
	router.HandleFunc("/units/edit", handler.HandleEditUnits).Methods("POST")
}

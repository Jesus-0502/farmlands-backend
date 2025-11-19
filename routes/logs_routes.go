package routes

import (
	"database/sql"
	"farmlands-backend/handlers"

	"github.com/gorilla/mux"
)

func RegisterLogsRoutes(router *mux.Router, db *sql.DB) {
	handler := handlers.NewLogsHandler(db)

	// router.HandleFunc("/farm_tasks", handler.HandleListFarmTasks).Methods("GET")
	router.HandleFunc("/log", handler.HandleAddLog).Methods("POST")
	router.HandleFunc("/log", handler.HandleSearchLog).Methods("GET")
	router.HandleFunc("/log/delete", handler.HandleDeleteLog).Methods("POST")
}

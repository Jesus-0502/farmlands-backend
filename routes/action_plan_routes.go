package routes

import (
	"database/sql"
	"farmlands-backend/handlers"

	"github.com/gorilla/mux"
)

func RegisterActionPlanRoutes(router *mux.Router, db *sql.DB) {
	handler := handlers.NewActionPlanHandler(db)

	router.HandleFunc("/action_plan", handler.HandleNewActionPlan).Methods("POST")
	router.HandleFunc("/action_plan", handler.HandleSearchActionPlan).Methods("GET")
	router.HandleFunc("/action_plan/edit", handler.HandleEditActionPlan).Methods("POST")
	router.HandleFunc("/action_plan/delete", handler.HandleDeleteActionPlan).Methods("POST")

}

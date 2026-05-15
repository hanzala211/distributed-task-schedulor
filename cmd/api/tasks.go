package main

import (
	"net/http"

	"github.com/hanzala211/go-backend-template/internal/models"
	"go.uber.org/zap"
)

func (app *application) addTask(w http.ResponseWriter, r *http.Request) {
	var req models.AddTaskAPIDTO
	err := app.DecodeStruct(w, r, &req)
	if err != nil {
		return
	}
	task, err := app.service.TaskService.InsertTask(r.Context(), &req)
	if err != nil {
		app.logger.Error("failed to add task", zap.Error(err))
		app.writeJSONError(w, http.StatusInternalServerError, "Server Error")
		return
	}
	app.writeJSON(w, http.StatusCreated, task)
}

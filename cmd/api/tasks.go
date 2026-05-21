package main

import (
	"errors"
	"net/http"

	"github.com/hanzala211/go-backend-template/internal/models"
	"github.com/hanzala211/go-backend-template/internal/service"
	"github.com/hanzala211/go-backend-template/internal/store"
	"go.uber.org/zap"
)

func (app *application) addTask(w http.ResponseWriter, r *http.Request) {
	var req models.AddTaskAPIDTO
	err := app.DecodeStruct(w, r, &req)
	if err != nil {
		return
	}
	task, err := app.service.Task.InsertTask(r.Context(), &req)
	if err != nil {
		var appErr *service.AppError
		if errors.As(err, &appErr) {
			if errors.Is(appErr.Err, store.ErrConflict) {
				app.logger.Error(appErr.Message, "targetURL", req.TargetURL)
				app.writeJSONError(w, http.StatusConflict, "A task with this configuration already exists")
				return
			} else if errors.Is(appErr.Err, store.ErrNotFound) {
				app.logger.Error(appErr.Message, "deps", req.Dependencies)
				app.writeJSONError(w, http.StatusBadRequest, "One or more dependent task do not exist")
				return
			} else {
				app.logger.Errorw("failed to add task", zap.Error(err))
				app.writeJSONError(w, http.StatusInternalServerError, "An unexpected server error occurred")
				return
			}
		}
	}
	app.writeJSON(w, http.StatusCreated, task)
}

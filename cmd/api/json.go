package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validator *validator.Validate

func init() {
	Validator = validator.New(validator.WithRequiredStructEnabled())
}

func (app *application) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.WriteHeader(statusCode)
	body := map[string]any{
		"status": "success",
		"data":   data,
	}
	json.NewEncoder(w).Encode(body)
}

func (app *application) writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	body := map[string]any{
		"status":  "error",
		"message": message,
	}
	json.NewEncoder(w).Encode(body)
}

func (app *application) DecodeStruct(w http.ResponseWriter, r *http.Request, req any) error {
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		fmtErr := fmt.Sprintf("Failed to decode request body: %v", err)
		app.writeJSONError(w, http.StatusBadRequest, fmtErr)
		return errors.New(fmtErr)
	}
	if err := Validator.Struct(req); err != nil {
		if errs, ok := err.(validator.ValidationErrors); ok {
			var messages []string
			for _, e := range errs {
				messages = append(messages, fmt.Sprintf("Field '%s' failed on the %s rule", e.Field(), e.Tag()))
			}
			app.writeJSONError(w, http.StatusBadRequest, strings.Join(messages, ", "))
			return err
		}
	}
	return nil
}

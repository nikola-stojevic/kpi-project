package utils

import (
	"encoding/json"
	"net/http"

	"kpiproject/models"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()
}

// DecodeAndValidate decodes the request body into a structure and validates it
func DecodeAndValidate(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		HandleMessageResponse(w, err.Error(), http.StatusBadRequest)
		return err
	}
	if err := Validate.Struct(v); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		errorMessages := make(map[string]string)

		for _, e := range validationErrors {
			errorMessages[e.Field()] = e.Tag()
		}
		HandleValidationResponse(w, http.StatusBadRequest, errorMessages)
		return err
	}
	return nil
}

// HandleAPIResponse handles both success and error responses
func HandleMessageResponse(w http.ResponseWriter, errorMessage string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	response := models.NewMessageResponse(statusCode, errorMessage)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// HandleValidationResponse handles validation errors response for struct validation
func HandleValidationResponse(w http.ResponseWriter, statusCode int, validationErrors interface{}) {
	w.Header().Set("Content-Type", "application/json")
	response := models.NewValidationResponse(statusCode, validationErrors)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// HandleDataResponse handles success responses with data
func HandleDataResponse(w http.ResponseWriter, message string, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	response := models.NewDataResponse(statusCode, message, data)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

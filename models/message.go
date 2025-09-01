package models

type MessageResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

type ValidationResponse struct {
	StatusCode int         `json:"status_code"`
	Errors     interface{} `json:"errors"`
}

type DataResponse struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
}

func NewMessageResponse(statusCode int, message string) MessageResponse {
	return MessageResponse{
		StatusCode: statusCode,
		Message:    message,
	}
}

func NewValidationResponse(statusCode int, errors interface{}) ValidationResponse {
	return ValidationResponse{
		StatusCode: statusCode,
		Errors:     errors,
	}
}

func NewDataResponse(statusCode int, message string, data interface{}) DataResponse {
	return DataResponse{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}
}

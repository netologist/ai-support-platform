package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	validator "github.com/go-playground/validator/v10"
)

type RequestValidator struct {
	validator *validator.Validate
}

func NewRequestValidator() RequestValidator {
	return RequestValidator{validator: validator.New()}
}

func (requestValidator RequestValidator) DecodeAndValidate(request *http.Request, destination any) (map[string][]string, error) {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return nil, fmt.Errorf("decode request body: %w", err)
	}

	if err := requestValidator.validator.Struct(destination); err != nil {
		fieldErrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return nil, err
		}

		validationErrors := make(map[string][]string, len(fieldErrors))
		for _, fieldError := range fieldErrors {
			fieldName := strings.ToLower(fieldError.Field())
			validationErrors[fieldName] = append(validationErrors[fieldName], validationMessage(fieldError))
		}

		return validationErrors, err
	}

	return nil, nil
}

func validationMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "uuid":
		return "must be a valid UUID"
	default:
		return "is invalid"
	}
}

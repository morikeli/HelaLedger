package validatorutil

import (
	"errors"
	"mime"
	"net/http"
)

func ValidateJSONContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return errors.New("invalid content type")
	}

	if mediaType != "application/json" {
		return errors.New("content type must be application/json")
	}

	return nil
}

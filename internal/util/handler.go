package util

import (
	"fmt"
	"net/http"
	"strconv"
)

func IntegerParam(r *http.Request, param string, defaultValue int) (int, error) {
	stringValue := r.URL.Query().Get(param)
	if stringValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(stringValue)
	if err != nil {
		return defaultValue, fmt.Errorf("parameter '%s' is not a valid integer", param)
	}

	return value, nil
}

package utils

import (
	"fmt"
	"net/http"
	"strconv"
)

const pageSizeDefault = 50
const pageSizeMax = 100

// GetPaginationParams calculates the offset and limit for pagination based on the provided values.
// If offset or limit are nil, default values are used. The limit is capped at a maximum value.
func GetPaginationParams(offset *int, limit *int) (int, int) {
	finalOffset := 0
	finalLimit := pageSizeDefault

	if offset != nil && *offset >= 0 {
		finalOffset = *offset
	}

	if limit != nil && *limit > 0 {
		finalLimit = min(*limit, pageSizeMax)
	}

	return finalOffset, finalLimit
}

// ParsePaginationParams extracts offset and limit from query parameters.
func ParsePaginationParams(r *http.Request) (*int, *int, error) {
	var offset, limit *int

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offsetVal, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid 'offset' query parameter, must be an integer")
		}
		if offsetVal < 0 {
			return nil, nil, fmt.Errorf("invalid 'offset' query parameter, must be a non-negative integer")
		}
		offset = &offsetVal
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limitVal, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid 'limit' query parameter, must be an integer")
		}
		if limitVal < 1 {
			return nil, nil, fmt.Errorf("invalid 'limit' query parameter, must be at least 1")
		}
		if limitVal > pageSizeMax {
			return nil, nil, fmt.Errorf("invalid 'limit' query parameter, maximum allowed is %d", pageSizeMax)
		}
		limit = &limitVal
	}

	return offset, limit, nil
}

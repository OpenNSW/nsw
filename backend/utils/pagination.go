package utils

const pageSizeDefault = 20
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

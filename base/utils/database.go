package utils

import "github.com/lib/pq"

func IsDBUniqueViolationError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}

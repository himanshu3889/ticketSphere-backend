package utils

import "github.com/lib/pq"

func IsPgDBUniqueViolationError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}

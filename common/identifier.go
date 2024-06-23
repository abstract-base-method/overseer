package common

import "github.com/google/uuid"

func GenerateUniqueId() string {
	return uuid.NewString()
}

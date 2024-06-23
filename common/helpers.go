package common

import (
	"crypto/sha512"
	"encoding/hex"
	"strings"
)

func GenerateRandomStringFromSeed(seeds ...string) string {
	var builder strings.Builder
	for _, seed := range seeds {
		builder.WriteString(seed)
		builder.WriteString("::")
	}
	sum := sha512.Sum512_256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])[:10]
}

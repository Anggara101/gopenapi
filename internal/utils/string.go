package utils

import (
	"strings"
)

func String(s string) *string {
	return &s
}

func CapitalizeFirstWord(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

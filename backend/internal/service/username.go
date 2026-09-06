package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxUsernameRunes = 100

func NormalizeAndValidateUsername(raw string) (string, error) {
	username := strings.TrimSpace(raw)
	if username == "" {
		return "", ErrUsernameRequired
	}
	if !utf8.ValidString(username) || utf8.RuneCountInString(username) > MaxUsernameRunes {
		return "", ErrUsernameInvalid
	}
	for _, r := range username {
		if unicode.IsControl(r) {
			return "", ErrUsernameInvalid
		}
	}
	return username, nil
}

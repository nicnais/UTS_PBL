package helper

import (
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

const PasswordCost = 12

func ValidEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email && len(email) <= 255 && strings.Contains(email, ".")
}

func PasswordError(password string) string {
	if utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return "Minimal 8 karakter dan maksimal 72 byte"
	}
	var letter, number bool
	for _, r := range password {
		letter = letter || unicode.IsLetter(r)
		number = number || unicode.IsDigit(r)
	}
	if !letter || !number {
		return "Harus mengandung huruf dan angka"
	}
	switch strings.ToLower(password) {
	case "password1", "password123", "12345678", "qwerty123", "rahasia123":
		return "Password terlalu umum"
	}
	return ""
}
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), PasswordCost)
	return string(hash), err
}
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

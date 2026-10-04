package helper

import (
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"testing"
	"time"
)

func TestPasswordRules(t *testing.T) {
	for _, p := range []string{"pendek1", "tanpaangka", "password123", strings.Repeat("a", 73)} {
		if PasswordError(p) == "" {
			t.Fatalf("password lemah diterima: %q", p)
		}
	}
	if msg := PasswordError("CucianBersih!2026"); msg != "" {
		t.Fatal(msg)
	}
}
func TestJWTValidation(t *testing.T) {
	secret := strings.Repeat("x", 32)
	m := NewJWT(secret, "laundry-api", time.Minute)
	good, err := m.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := m.Verify(good); err != nil || id != 42 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if _, err := NewJWT(strings.Repeat("z", 32), "laundry-api", time.Minute).Verify(good); err == nil {
		t.Fatal("signature salah diterima")
	}
	if _, err := NewJWT(secret, "issuer-lain", time.Minute).Verify(good); err == nil {
		t.Fatal("issuer salah diterima")
	}
	claims := jwt.RegisteredClaims{Issuer: "laundry-api", Subject: "42", Audience: jwt.ClaimStrings{"laundry-api"}}
	missing, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if _, err := m.Verify(missing); err == nil {
		t.Fatal("tanpa exp diterima")
	}
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := m.Verify(none); err == nil {
		t.Fatal("alg none diterima")
	}
	expired, _ := NewJWT(secret, "laundry-api", -time.Minute).Issue(42)
	if _, err := m.Verify(expired); err == nil {
		t.Fatal("token kedaluwarsa diterima")
	}
}

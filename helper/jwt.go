package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret []byte
	issuer string
	TTL    time.Duration
}

func NewJWT(secret, issuer string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), issuer: issuer, TTL: ttl}
}
func (m *JWTManager) Issue(userID int64) (string, error) {
	now := time.Now()
	id, err := RandomToken()
	if err != nil {
		return "", err
	}
	claims := jwt.RegisteredClaims{
		Issuer: m.issuer, Subject: strconv.FormatInt(userID, 10), Audience: jwt.ClaimStrings{"laundry-api"},
		IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.TTL)), ID: id,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}
func (m *JWTManager) Verify(raw string) (int64, error) {
	claims := new(jwt.RegisteredClaims)
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(m.issuer), jwt.WithAudience("laundry-api"),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, Error(401, "TOKEN_EXPIRED", "Access token kedaluwarsa")
		}
		return 0, Error(401, "INVALID_TOKEN", "Access token tidak valid")
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id < 1 || !token.Valid {
		return 0, Error(401, "INVALID_TOKEN", "Access token tidak valid")
	}
	return id, nil
}
func RandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func TokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
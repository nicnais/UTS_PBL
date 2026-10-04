package service

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"laundry-api/app/model"
	"laundry-api/app/repository"
	"laundry-api/helper"
)

type AuthService struct {
	store      repository.Store
	jwt        *helper.JWTManager
	refreshTTL time.Duration
	dummyHash  string
}

func NewAuth(store repository.Store, jwt *helper.JWTManager, ttl time.Duration) (*AuthService, error) {
	// Username yang tidak terdaftar tetap melewati bcrypt untuk mengurangi perbedaan waktu.
	dummy, err := helper.HashPassword("DummyPasswordForTiming942!")
	if err != nil {
		return nil, err
	}
	return &AuthService{store: store, jwt: jwt, refreshTTL: ttl, dummyHash: dummy}, nil
}
func (s *AuthService) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	fields := map[string]string{}
	if n := utf8.RuneCountInString(req.Name); n < 2 || n > 100 {
		fields["name"] = "Panjang nama harus 2 sampai 100 karakter"
	}
	if !helper.ValidEmail(req.Email) {
		fields["email"] = "Format email tidak valid"
	}
	if msg := helper.PasswordError(req.Password); msg != "" {
		fields["password"] = msg
	}
	if len(fields) > 0 {
		return helper.Validation(fields)
	}
	hash, err := helper.HashPassword(req.Password)
	if err != nil {
		return err
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	user, err := s.store.CreateUser(ctx, req.Name, req.Email, hash, "customer")
	if err != nil {
		return err
	}
	return helper.OK(c, 201, "Pendaftaran berhasil", user)
}
func (s *AuthService) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	if req.Email == "" || req.Password == "" {
		return helper.Validation(map[string]string{"credentials": "Email dan password wajib diisi"})
	}
	if len(req.Email) > 255 || len(req.Password) > 72 {
		return helper.Error(401, "INVALID_CREDENTIALS", "Email atau password salah")
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	user, err := s.store.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}
	hash := user.PasswordHash
	if errors.Is(err, model.ErrNotFound) {
		hash = s.dummyHash
	}
	valid := helper.CheckPassword(hash, req.Password)
	if !valid || user.ID == 0 {
		return helper.Error(401, "INVALID_CREDENTIALS", "Email atau password salah")
	}
	access, err := s.jwt.Issue(user.ID)
	if err != nil {
		return err
	}
	refresh, err := helper.RandomToken()
	if err != nil {
		return err
	}
	if err = s.store.SaveToken(ctx, user.ID, helper.TokenHash(refresh), time.Now().Add(s.refreshTTL)); err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return helper.OK(c, 200, "Login berhasil", model.Tokens{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: int64(s.jwt.TTL.Seconds())})
}
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	var req model.RefreshRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	if len(req.RefreshToken) != 43 {
		return model.ErrInvalidToken
	}
	replacement, err := helper.RandomToken()
	if err != nil {
		return err
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	user, err := s.store.RotateToken(ctx, helper.TokenHash(req.RefreshToken), helper.TokenHash(replacement), time.Now().Add(s.refreshTTL))
	if err != nil {
		return err
	}
	access, err := s.jwt.Issue(user.ID)
	if err != nil {
		return err
	}
	c.Set("Cache-Control", "no-store")
	return helper.OK(c, 200, "Token diperbarui", model.Tokens{AccessToken: access, RefreshToken: replacement, TokenType: "Bearer", ExpiresIn: int64(s.jwt.TTL.Seconds())})
}
func (s *AuthService) Logout(c *fiber.Ctx) error {
	var req model.RefreshRequest
	if err := helper.Decode(c, &req); err != nil {
		return err
	}
	if len(req.RefreshToken) != 43 {
		return helper.Validation(map[string]string{"refresh_token": "Format refresh token tidak valid"})
	}
	ctx, cancel := helper.DBContext(c)
	defer cancel()
	if err := s.store.RevokeToken(ctx, helper.TokenHash(req.RefreshToken)); err != nil {
		return err
	}
	return c.SendStatus(204)
}
func (s *AuthService) Me(c *fiber.Ctx) error {
	return helper.OK(c, 200, "Profil pengguna", helper.CurrentUser(c))
}

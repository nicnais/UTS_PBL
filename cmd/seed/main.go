package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"laundry-api/app/model"
	"laundry-api/app/repository"
	"laundry-api/config"
	"laundry-api/database"
	"laundry-api/helper"
)

func run() error {
	if err := config.LoadEnv(); err != nil {
		return err
	}
	name := strings.TrimSpace(config.GetEnv("SEED_STAFF_NAME", "Petugas Laundry"))
	email := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_STAFF_EMAIL")))
	password := os.Getenv("SEED_STAFF_PASSWORD")
	if n := utf8.RuneCountInString(name); n < 2 || n > 100 {
		return fmt.Errorf("nama petugas harus 2 sampai 100 karakter")
	}
	if !helper.ValidEmail(email) {
		return fmt.Errorf("isi SEED_STAFF_EMAIL dengan email yang valid")
	}
	if msg := helper.PasswordError(password); msg != "" {
		return fmt.Errorf("SEED_STAFF_PASSWORD: %s", msg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.NewPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := repository.New(pool)
	existing, err := store.UserByEmail(ctx, email)
	if err == nil {
		if existing.Role != "staff" {
			return fmt.Errorf("email sudah dipakai pelanggan; gunakan email petugas lain (akun tidak diubah)")
		}
		fmt.Println("Akun petugas sudah tersedia; password tidak diubah.")
		return nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return err
	}
	hash, err := helper.HashPassword(password)
	if err != nil {
		return err
	}
	user, err := store.CreateUser(ctx, name, email, hash, "staff")
	if err != nil {
		return err
	}
	fmt.Printf("Akun petugas dibuat: %s (id=%d)\n", user.Email, user.ID)
	return nil
}
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

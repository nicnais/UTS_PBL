package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"laundry-api/app/repository"
	"laundry-api/config"
	"laundry-api/helper"
)

func TestLaundryAPI(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("Set TEST_DATABASE_URL ke database pengujian untuk menjalankan integration test")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("laundry_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, file := range []string{"001_init.sql", "002_seed.sql"} {
		sql, err := os.ReadFile(filepath.Join("..", "database", "migrations", file))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	settings := config.Settings{Secret: strings.Repeat("integration-secret-", 3), Issuer: "laundry-api", AccessTTL: time.Minute, RefreshTTL: time.Hour, AllowedOrigins: "http://localhost:5173"}
	app, err := config.NewApp(pool, settings)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown()
	call := func(method, path, token string, body any, want int) map[string]any {
		t.Helper()
		var content []byte
		if body != nil {
			content, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req, _ := http.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(content))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%s %s: want %d got %d body=%s", method, path, want, resp.StatusCode, raw)
		}
		result := map[string]any{}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
		}
		return result
	}
	data := func(r map[string]any) map[string]any { return r["data"].(map[string]any) }
	call("GET", "/health", "", nil, 200)
	call("GET", "/orders", "", nil, 401)
	a := data(call("POST", "/auth/register", "", map[string]any{"name": "Customer A", "email": "a@example.com", "password": "LaundryStrong123!"}, 201))
	b := data(call("POST", "/auth/register", "", map[string]any{"name": "Customer B", "email": "b@example.com", "password": "LaundryStrong123!"}, 201))
	if a["role"] != "customer" || a["password_hash"] != nil || a["password"] != nil {
		t.Fatalf("unsafe registration: %v", a)
	}
	call("POST", "/auth/register", "", map[string]any{"name": "Escalation", "email": "evil@example.com", "password": "LaundryStrong123!", "role": "staff"}, 400)
	call("POST", "/auth/register", "", map[string]any{"name": "Duplicate", "email": "A@example.com", "password": "LaundryStrong123!"}, 409)
	hash, err := helper.HashPassword("LaundryStrong123!")
	if err != nil {
		t.Fatal(err)
	}
	store := repository.New(pool)
	staff, err := store.CreateUser(ctx, "Petugas", "staff@example.com", hash, "staff")
	if err != nil {
		t.Fatal(err)
	}
	login := func(email string) map[string]any {
		return data(call("POST", "/auth/login", "", map[string]any{"email": email, "password": "LaundryStrong123!"}, 200))
	}
	tokenA := login("a@example.com")
	tokenB := login("b@example.com")
	tokenS := login("staff@example.com")
	at := tokenA["access_token"].(string)
	bt := tokenB["access_token"].(string)
	st := tokenS["access_token"].(string)
	call("GET", "/auth/me", at, nil, 200)
	call("POST", "/services", at, map[string]any{"name": "Forbidden", "price_per_kg": 7000}, 403)
	call("POST", "/services", st, map[string]any{"name": "Express", "price_per_kg": 12000}, 201)
	call("POST", "/services", st, map[string]any{"name": "Bad price", "price_per_kg": 0}, 422)
	call("POST", "/orders", at, map[string]any{"service_id": 1, "customer_id": int64(b["id"].(float64))}, 400)
	order := data(call("POST", "/orders", at, map[string]any{"service_id": 1, "notes": "Pisahkan putih"}, 201))
	path := fmt.Sprintf("/orders/%.0f", order["id"])
	if order["customer_id"] != a["id"] || order["total_price"] != nil {
		t.Fatalf("bad order: %v", order)
	}
	call("GET", path, bt, nil, 403)
	list := call("GET", "/orders?customer_id=1", bt, nil, 200)
	if len(list["data"].([]any)) != 0 {
		t.Fatal("list leaks another customer's order")
	}
	call("PATCH", path, at, map[string]any{"status": "washing"}, 403)
	call("PATCH", path, st, map[string]any{"weight_kg": -1}, 422)
	call("PATCH", path, st, map[string]any{"weight_kg": nil}, 422)
	call("PATCH", path, st, map[string]any{"total_price": 1}, 400)
	call("PATCH", path, st, map[string]any{"status": "washing"}, 409)
	order = data(call("PATCH", path, st, map[string]any{"weight_kg": 3}, 200))
	if order["total_price"] != float64(21000) || order["weight_kg"] != "3.00" {
		t.Fatalf("wrong price: %v", order)
	}
	call("PATCH", "/services/1", st, map[string]any{"price_per_kg": 9000}, 200)
	order = data(call("PATCH", path, st, map[string]any{"weight_kg": 4}, 200))
	if order["total_price"] != float64(28000) || order["price_per_kg_snapshot"] != float64(7000) {
		t.Fatalf("snapshot changed: %v", order)
	}
	call("PATCH", path, st, map[string]any{"status": "completed"}, 409)
	call("PATCH", path, st, map[string]any{"status": "washing"}, 200)
	call("PATCH", path, st, map[string]any{"weight_kg": 5}, 409)
	call("PATCH", path, st, map[string]any{"status": "ready"}, 200)
	call("PATCH", path, st, map[string]any{"status": "completed"}, 409)
	call("PATCH", path, st, map[string]any{"status": "completed", "payment_status": "paid"}, 200)
	call("PATCH", path, st, map[string]any{"status": "completed"}, 200)
	call("PATCH", path, st, map[string]any{"payment_status": "unpaid"}, 409)
	call("PATCH", "/services/1", st, map[string]any{"is_active": false}, 200)
	call("POST", "/orders", at, map[string]any{"service_id": 1}, 404)
	call("GET", path, st, nil, 200)
	call("GET", "/orders?status=completed&limit=1", st, nil, 200)
	call("GET", "/orders?page=-1", at, nil, 422)
	call("GET", "/orders?limit=101", at, nil, 422)

	// Rotasi bersamaan: hanya satu request yang boleh berhasil.
	raw, _ := json.Marshal(map[string]string{"refresh_token": tokenA["refresh_token"].(string)})
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("POST", "http://localhost/api/v1/auth/refresh", bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				statuses <- 0
				return
			}
			defer resp.Body.Close()
			statuses <- resp.StatusCode
		}()
	}
	wg.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[200] != 1 || counts[401] != 1 {
		t.Fatalf("refresh race: %v", counts)
	}
	call("POST", "/auth/refresh", "", map[string]any{"refresh_token": tokenA["refresh_token"]}, 401)
	rotated := data(call("POST", "/auth/refresh", "", map[string]any{"refresh_token": tokenB["refresh_token"]}, 200))
	call("POST", "/auth/logout", "", map[string]any{"refresh_token": rotated["refresh_token"]}, 204)
	call("POST", "/auth/refresh", "", map[string]any{"refresh_token": rotated["refresh_token"]}, 401)
	// Logout mencabut refresh token, bukan access token yang belum expired.
	call("GET", "/auth/me", bt, nil, 200)
	// Perubahan permission berlaku pada request berikutnya, JWT tidak menyimpan role.
	if _, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE role_id=(SELECT role_id FROM users WHERE id=$1) AND permission_id=(SELECT id FROM permissions WHERE name='service:create')`, staff.ID); err != nil {
		t.Fatal(err)
	}
	call("POST", "/services", st, map[string]any{"name": "Revoked", "price_per_kg": 7000}, 403)
	failed1 := call("POST", "/auth/login", "", map[string]any{"email": "missing@example.com", "password": "WrongPassword123!"}, 401)
	failed2 := call("POST", "/auth/login", "", map[string]any{"email": "a@example.com", "password": "WrongPassword123!"}, 401)
	if failed1["message"] != failed2["message"] {
		t.Fatal("login enumerates users")
	}
	call("POST", "/auth/login", "", map[string]any{"email": "a@example.com", "password": "WrongPassword123!"}, 429)
	var stored string
	if err := pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE email='a@example.com'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "$2") || stored == "LaundryStrong123!" {
		t.Fatal("password not hashed")
	}
	if err := pool.QueryRow(ctx, `SELECT token_hash FROM refresh_tokens LIMIT 1`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 64 {
		t.Fatal("refresh token not stored as sha256 hash")
	}
}
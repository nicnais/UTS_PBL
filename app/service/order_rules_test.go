package service

import (
	"encoding/json"
	"laundry-api/app/model"
	"testing"
)

func ptr[T any](v T) *T { return &v }
func TestOrderRules(t *testing.T) {
	base := model.Order{ID: 1, CustomerID: 7, Status: "pending", PaymentStatus: "unpaid"}
	t.Run("hitung tepat tanpa floating point", func(t *testing.T) {
		c, err := DecideOrderChange(base, 7000, model.PatchOrderRequest{WeightKg: ptr(json.Number("1.23"))})
		if err != nil || *c.TotalPrice != 8610 || *c.WeightKg != "1.23" {
			t.Fatalf("hasil=%+v err=%v", c, err)
		}
	})
	t.Run("pembulatan rupiah", func(t *testing.T) {
		c, err := DecideOrderChange(base, 1050, model.PatchOrderRequest{WeightKg: ptr(json.Number("0.01"))})
		if err != nil || *c.TotalPrice != 11 {
			t.Fatalf("hasil=%+v err=%v", c, err)
		}
	})
	t.Run("snapshot tarif dipertahankan", func(t *testing.T) {
		o := base
		o.PricePerKgSnapshot = ptr(int64(7000))
		c, err := DecideOrderChange(o, 12000, model.PatchOrderRequest{WeightKg: ptr(json.Number("3"))})
		if err != nil || *c.TotalPrice != 21000 {
			t.Fatalf("hasil=%+v err=%v", c, err)
		}
	})
	for _, bad := range []string{"0", "-1", "1.234", "1e3", "1000.01"} {
		t.Run("berat_"+bad, func(t *testing.T) {
			if _, _, err := ParseWeight(bad); err == nil {
				t.Fatal("berat invalid diterima")
			}
		})
	}
	t.Run("tolak lompatan status", func(t *testing.T) {
		if _, err := DecideOrderChange(base, 7000, model.PatchOrderRequest{Status: ptr("completed")}); err == nil {
			t.Fatal("lompatan diterima")
		}
	})
	t.Run("tolak washing tanpa berat", func(t *testing.T) {
		if _, err := DecideOrderChange(base, 7000, model.PatchOrderRequest{Status: ptr("washing")}); err == nil {
			t.Fatal("washing tanpa berat diterima")
		}
	})
	t.Run("harus lunas sebelum selesai", func(t *testing.T) {
		o := base
		o.Status = "ready"
		o.WeightKg = ptr("3.00")
		o.TotalPrice = ptr(int64(21000))
		o.PricePerKgSnapshot = ptr(int64(7000))
		if _, err := DecideOrderChange(o, 7000, model.PatchOrderRequest{Status: ptr("completed")}); err == nil {
			t.Fatal("selesai tanpa lunas")
		}
		if _, err := DecideOrderChange(o, 7000, model.PatchOrderRequest{Status: ptr("completed"), PaymentStatus: ptr("paid")}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("berat tidak berubah setelah lunas", func(t *testing.T) {
		o := base
		o.PaymentStatus = "paid"
		if _, err := DecideOrderChange(o, 7000, model.PatchOrderRequest{WeightKg: ptr(json.Number("2"))}); err == nil {
			t.Fatal("berat berubah setelah lunas")
		}
	})
}
func TestOwnership(t *testing.T) {
	o := model.Order{CustomerID: 7}
	if CanReadOrder(model.User{ID: 8}, o) {
		t.Fatal("pelanggan lain dapat membaca")
	}
	if !CanReadOrder(model.User{ID: 7}, o) {
		t.Fatal("pemilik ditolak")
	}
	if !CanReadOrder(model.User{ID: 8, Permissions: []string{"order:read:any"}}, o) {
		t.Fatal("petugas ditolak")
	}
	if CanReadOrder(model.User{}, model.Order{}) {
		t.Fatal("identitas kosong diterima")
	}
}

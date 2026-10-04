package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"laundry-api/app/model"
	"laundry-api/helper"
)

var weightPattern = regexp.MustCompile(`^[0-9]{1,4}(\.[0-9]{1,2})?$`)

// Berat dikonversi ke satuan 0,01 kg agar perhitungan uang tidak memakai float.
func ParseWeight(value string) (int64, string, error) {
	if !weightPattern.MatchString(value) {
		return 0, "", helper.Validation(map[string]string{"weight_kg": "Gunakan angka positif dengan maksimal 2 desimal"})
	}
	parts := strings.SplitN(value, ".", 2)
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	fraction := "00"
	if len(parts) == 2 {
		fraction = (parts[1] + "00")[:2]
	}
	decimal, _ := strconv.ParseInt(fraction, 10, 64)
	hundredths := whole*100 + decimal
	if hundredths < 1 || hundredths > 100000 {
		return 0, "", helper.Validation(map[string]string{"weight_kg": "Berat harus 0.01 sampai 1000 kg"})
	}
	return hundredths, fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100), nil
}
func ValidStatus(status string) bool {
	return status == "pending" || status == "washing" || status == "ready" || status == "completed"
}
func CanReadOrder(user model.User, order model.Order) bool {
	return user.ID > 0 && (user.ID == order.CustomerID || user.Can("order:read:any"))
}

// Fungsi murni: dapat diuji tanpa Fiber atau koneksi database.
func DecideOrderChange(current model.Order, rate int64, req model.PatchOrderRequest) (model.OrderChange, error) {
	next := model.OrderChange{WeightKg: current.WeightKg, PricePerKgSnapshot: current.PricePerKgSnapshot, TotalPrice: current.TotalPrice, Status: current.Status, PaymentStatus: current.PaymentStatus}
	if req.WeightKg == nil && req.Status == nil && req.PaymentStatus == nil {
		return next, helper.Validation(map[string]string{"body": "Minimal satu field harus dikirim"})
	}
	if req.WeightKg != nil {
		if current.Status != "pending" {
			return next, helper.BadState("Berat hanya boleh diubah ketika pending")
		}
		hundredths, normalized, err := ParseWeight(req.WeightKg.String())
		if err != nil {
			return next, err
		}
		if current.PricePerKgSnapshot != nil {
			rate = *current.PricePerKgSnapshot
		}
		if rate <= 0 || rate > 1000000000 {
			return next, helper.BadState("Tarif layanan tidak valid")
		}
		total := (hundredths*rate + 50) / 100 // Pembulatan setengah ke atas ke rupiah penuh.
		if total < 1 {
			return next, helper.Validation(map[string]string{"weight_kg": "Total biaya harus minimal Rp1"})
		}
		next.WeightKg = &normalized
		next.PricePerKgSnapshot = &rate
		next.TotalPrice = &total
	}
	if req.PaymentStatus != nil {
		if *req.PaymentStatus != "paid" && *req.PaymentStatus != "unpaid" {
			return next, helper.Validation(map[string]string{"payment_status": "Pilih unpaid atau paid"})
		}
		if current.PaymentStatus == "paid" && *req.PaymentStatus == "unpaid" {
			return next, helper.BadState("Pembayaran lunas tidak dapat dibatalkan melalui endpoint ini")
		}
		if *req.PaymentStatus == "paid" && next.TotalPrice == nil {
			return next, helper.BadState("Tentukan berat dan biaya sebelum mencatat pembayaran")
		}
		next.PaymentStatus = *req.PaymentStatus
	}
	// Setelah lunas, nominal tidak boleh berubah walaupun status masih pending.
	if req.WeightKg != nil && current.PaymentStatus == "paid" {
		return next, helper.BadState("Berat tidak boleh diubah setelah pembayaran lunas")
	}
	if req.Status != nil {
		wanted := *req.Status
		if !ValidStatus(wanted) {
			return next, helper.Validation(map[string]string{"status": "Status tidak dikenal"})
		}
		transitions := map[string]string{"pending": "washing", "washing": "ready", "ready": "completed"}
		if wanted != current.Status && transitions[current.Status] != wanted {
			return next, helper.BadState("Urutan status harus pending -> washing -> ready -> completed")
		}
		next.Status = wanted
	}
	if next.Status != "pending" && (next.WeightKg == nil || next.TotalPrice == nil || next.PricePerKgSnapshot == nil) {
		return next, helper.BadState("Berat dan biaya harus ditetapkan sebelum mencuci")
	}
	if next.Status == "completed" && next.PaymentStatus != "paid" {
		return next, helper.BadState("Pesanan harus lunas sebelum diselesaikan")
	}
	return next, nil
}

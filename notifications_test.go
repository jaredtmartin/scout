package scout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewFlash(t *testing.T) {
	flashes := NewFlash()
	if flashes == nil {
		t.Fatal("expected NewFlash to return a non-nil pointer")
	}
	if len(*flashes) != 0 {
		t.Errorf("expected empty flashes, got length %d", len(*flashes))
	}
}

func TestFlashes_Add(t *testing.T) {
	flashes := NewFlash()

	// Add Success flash without expiry
	flashes.Success("success message")
	if len(*flashes) != 1 {
		t.Fatalf("expected 1 flash, got %d", len(*flashes))
	}
	f1 := (*flashes)[0]
	if f1.Message != "success message" {
		t.Errorf("expected Message 'success message', got %q", f1.Message)
	}
	if f1.Urgency != SuccessUrgency {
		t.Errorf("expected Urgency %q, got %q", SuccessUrgency, f1.Urgency)
	}
	if f1.Expiry != defaultExpiry {
		t.Errorf("expected Expiry %d, got %d", defaultExpiry, f1.Expiry)
	}
	if f1.Id <= 0 {
		t.Errorf("expected ID to be positive, got %d", f1.Id)
	}

	// Add Error flash with expiry
	time.Sleep(time.Millisecond) // ensure a different timestamp
	flashes.Error("error message", 10)
	if len(*flashes) != 2 {
		t.Fatalf("expected 2 flashes, got %d", len(*flashes))
	}
	f2 := (*flashes)[1]
	if f2.Message != "error message" {
		t.Errorf("expected Message 'error message', got %q", f2.Message)
	}
	if f2.Urgency != ErrorUrgency {
		t.Errorf("expected Urgency %q, got %q", ErrorUrgency, f2.Urgency)
	}
	if f2.Expiry != 10 {
		t.Errorf("expected Expiry 10, got %d", f2.Expiry)
	}
	if f2.Id <= f1.Id {
		t.Errorf("expected unique incremental IDs, got f1.Id=%d, f2.Id=%d", f1.Id, f2.Id)
	}

	// Add Info and Warning flashes
	flashes.Info("info message")
	flashes.Warning("warning message")
	if len(*flashes) != 4 {
		t.Fatalf("expected 4 flashes, got %d", len(*flashes))
	}
	if (*flashes)[2].Urgency != InfoUrgency {
		t.Errorf("expected third flash to be InfoUrgency, got %q", (*flashes)[2].Urgency)
	}
	if (*flashes)[3].Urgency != WarningUrgency {
		t.Errorf("expected fourth flash to be WarningUrgency, got %q", (*flashes)[3].Urgency)
	}
}

func TestFlashes_Clear(t *testing.T) {
	flashes := NewFlash()
	flashes.Success("msg")
	if len(*flashes) != 1 {
		t.Fatalf("expected 1 flash before clear, got %d", len(*flashes))
	}

	flashes.Clear(nil)
	if len(*flashes) != 0 {
		t.Errorf("expected 0 flashes after clear, got %d", len(*flashes))
	}
}

func TestFlashes_EncodeDecode(t *testing.T) {
	flashes := NewFlash()
	flashes.Success("hello")
	flashes.Error("world", 30)

	encoded, err := flashes.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("expected encoded data, got empty byte slice")
	}

	// Decode back
	decoded := NewFlash()
	err = decoded.Decode(string(encoded))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(*decoded) != 2 {
		t.Fatalf("expected 2 decoded flashes, got %d", len(*decoded))
	}
	if (*decoded)[0].Message != "hello" || (*decoded)[0].Urgency != SuccessUrgency {
		t.Errorf("first decoded flash does not match original: %+v", (*decoded)[0])
	}
	if (*decoded)[1].Message != "world" || (*decoded)[1].Urgency != ErrorUrgency || (*decoded)[1].Expiry != 30 {
		t.Errorf("second decoded flash does not match original: %+v", (*decoded)[1])
	}

	// Test Decode with raw JSON instead of base64 encoded JSON
	rawJSON := `[{"Id":123,"Message":"raw json","Urgency":"info","Expiry":5}]`
	decodedRaw := NewFlash()
	err = decodedRaw.Decode(rawJSON)
	if err != nil {
		t.Fatalf("Decode of raw JSON failed: %v", err)
	}
	if len(*decodedRaw) != 1 {
		t.Fatalf("expected 1 decoded flash from raw JSON, got %d", len(*decodedRaw))
	}
	if (*decodedRaw)[0].Message != "raw json" || (*decodedRaw)[0].Id != 123 || (*decodedRaw)[0].Urgency != InfoUrgency {
		t.Errorf("decoded flash from raw JSON does not match: %+v", (*decodedRaw)[0])
	}

	// Test Decode with invalid input
	invalidDecoded := NewFlash()
	err = invalidDecoded.Decode("invalid-base64-and-invalid-json-{}")
	if err == nil {
		t.Error("expected error decoding invalid string, got nil")
	}
}

func TestFlashes_Save(t *testing.T) {
	flashes := NewFlash()
	flashes.Success("test save")

	rec := httptest.NewRecorder()
	err := flashes.Save(rec)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	res := rec.Result()
	cookies := res.Cookies()
	var flashCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == FlashCookieName {
			flashCookie = c
			break
		}
	}

	if flashCookie == nil {
		t.Fatal("expected flash cookie to be set")
	}

	if flashCookie.Path != "/" {
		t.Errorf("expected cookie path '/', got %q", flashCookie.Path)
	}
	if flashCookie.HttpOnly != true {
		t.Errorf("expected HttpOnly cookie, got %t", flashCookie.HttpOnly)
	}
	if flashCookie.Secure != true {
		t.Errorf("expected Secure cookie, got %t", flashCookie.Secure)
	}
	if flashCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSiteLaxMode cookie, got %v", flashCookie.SameSite)
	}

	// Verify the value by decoding it
	decoded := NewFlash()
	err = decoded.Decode(flashCookie.Value)
	if err != nil {
		t.Fatalf("failed to decode saved cookie value: %v", err)
	}
	if len(*decoded) != 1 || (*decoded)[0].Message != "test save" {
		t.Errorf("decoded saved flashes did not match: %+v", *decoded)
	}
}

func TestLoadFlashes(t *testing.T) {
	// Case 1: No cookie
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	flashes := loadFlashes(w, r)
	if len(*flashes) != 0 {
		t.Errorf("expected 0 flashes when no cookie, got %d", len(*flashes))
	}
	if len(w.Result().Cookies()) != 0 {
		t.Errorf("expected no cookies set when loading empty/no flashes")
	}

	// Case 2: Valid flash cookie exists
	originalFlashes := NewFlash()
	originalFlashes.Success("message 1")
	encoded, err := originalFlashes.Encode()
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	r = httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  FlashCookieName,
		Value: string(encoded),
	})
	w = httptest.NewRecorder()

	loaded := loadFlashes(w, r)
	if len(*loaded) != 1 {
		t.Fatalf("expected 1 loaded flash, got %d", len(*loaded))
	}
	if (*loaded)[0].Message != "message 1" {
		t.Errorf("expected message 'message 1', got %q", (*loaded)[0].Message)
	}

	// Verify that the cookie was consumed/cleared in the response
	cookies := w.Result().Cookies()
	var flashCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == FlashCookieName {
			flashCookie = c
			break
		}
	}
	if flashCookie == nil {
		t.Fatal("expected flash cookie to be cleared in the response")
	}
	if flashCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge to be -1, got %d", flashCookie.MaxAge)
	}

	// Case 3: Invalid cookie data
	r = httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{
		Name:  FlashCookieName,
		Value: "invalid-cookie-data",
	})
	w = httptest.NewRecorder()

	loadedInvalid := loadFlashes(w, r)
	if len(*loadedInvalid) != 0 {
		t.Errorf("expected 0 flashes when cookie is invalid, got %d", len(*loadedInvalid))
	}
	// Verify that the invalid cookie was not consumed/cleared (per early return implementation)
	for _, c := range w.Result().Cookies() {
		if c.Name == FlashCookieName {
			t.Errorf("did not expect flash cookie to be cleared on decode failure, but got cookie: %+v", c)
		}
	}
}

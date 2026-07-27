package scout

import (
	"encoding/json"
	"net/http"
	"time"
)

const FlashCookieName = "flash"

type Urgency string

const (
	SuccessUrgency Urgency = "Success"
	ErrorUrgency   Urgency = "Error"
	InfoUrgency    Urgency = "Info"
	WarningUrgency Urgency = "Warning"
)

type Flash struct {
	Id      int64
	Message string
	Urgency Urgency
	Expiry  int
}
type FlashCollection []Flash

func NewFlash() *FlashCollection {
	return &FlashCollection{}
}
func GetFlash(w http.ResponseWriter, r *http.Request) (*FlashCollection, error) {
	cookie, err := r.Cookie(FlashCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return NewFlash(), nil
		}
		return nil, err
	}
	flash := NewFlash()
	if err := flash.Decode(cookie.Value); err != nil {
		return nil, err
	}
	// Consume urgent messages by removing the cookie
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
	return flash, nil
}
func (f *FlashCollection) Add(msg string, ugency Urgency, expiry int) {
	(*f) = append((*f), Flash{
		Id:      time.Now().UnixNano(),
		Message: msg,
		Urgency: ugency,
		Expiry:  expiry,
	})
}
func (f *FlashCollection) Success(msg string) {
	f.Add(msg, SuccessUrgency, 0)
}

func (f *FlashCollection) Error(msg string) {
	f.Add(msg, ErrorUrgency, 0)
}

func (f *FlashCollection) Info(msg string) {
	f.Add(msg, InfoUrgency, 0)
}

func (f *FlashCollection) Warning(msg string) {
	f.Add(msg, WarningUrgency, 0)
}

func (f *FlashCollection) Save(w http.ResponseWriter) error {
	encoded, err := f.Encode()
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     FlashCookieName,
		Value:    string(encoded),
		Path:     "/",
		Expires:  time.Now().AddDate(1, 0, 0),
		MaxAge:   60 * 60 * 24 * 365,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	return nil
}
func (f *FlashCollection) Clear(w http.ResponseWriter) {
	*f = FlashCollection{}
}

// Encode the flash struct to base64 to store in a cookie
func (f *FlashCollection) Encode() ([]byte, error) {
	// Simple JSON encoding for now, can be improved
	return json.Marshal(*f)
}

// Decode the flash struct from base64 from a cookie
func (f *FlashCollection) Decode(encoded string) error {
	return json.Unmarshal([]byte(encoded), f)
}

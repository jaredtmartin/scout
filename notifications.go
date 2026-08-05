package scout

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

const FlashCookieName = "flash"

type Urgency string

const defaultExpiry = 5000
const (
	SuccessUrgency Urgency = "success"
	ErrorUrgency   Urgency = "error"
	InfoUrgency    Urgency = "info"
	WarningUrgency Urgency = "warning"
)

type Flash struct {
	Id      int64
	Message string
	Urgency Urgency
	Expiry  int
}
type Flashes []Flash

func NewFlash() *Flashes {
	return &Flashes{}
}
func loadFlashes(w http.ResponseWriter, r *http.Request) *Flashes {
	cookie, err := r.Cookie(FlashCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return NewFlash()
		}
		return NewFlash()
	}
	flash := NewFlash()
	if err := flash.Decode(cookie.Value); err != nil {
		return NewFlash()
	}
	// Consume urgent messages by removing the cookie
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
	return flash
}
func (f *Flashes) Add(msg string, ugency Urgency, expiry ...int) {
	var expiryInt int = defaultExpiry
	if len(expiry) > 0 {
		expiryInt = expiry[0]
	}
	(*f) = append((*f), Flash{
		Id:      time.Now().UnixNano(),
		Message: msg,
		Urgency: ugency,
		Expiry:  expiryInt,
	})
}
func (f *Flashes) Success(msg string, expiry ...int) {
	f.Add(msg, SuccessUrgency, expiry...)
}

func (f *Flashes) Error(msg string, expiry ...int) {
	f.Add(msg, ErrorUrgency, expiry...)
}

func (f *Flashes) Info(msg string, expiry ...int) {
	f.Add(msg, InfoUrgency, expiry...)
}

func (f *Flashes) Warning(msg string, expiry ...int) {
	f.Add(msg, WarningUrgency, expiry...)
}
func (f *Flashes) Save(w http.ResponseWriter) error {
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
func (f *Flashes) Clear(w http.ResponseWriter) {
	*f = Flashes{}
}

// Encode the flash struct to base64 to store in a cookie
func (f *Flashes) Encode() ([]byte, error) {
	payload, err := json.Marshal(*f)
	if err != nil {
		return nil, err
	}
	return []byte(base64.StdEncoding.EncodeToString(payload)), nil
}

// Decode the flash struct from a cookie value
func (f *Flashes) Decode(encoded string) error {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		return json.Unmarshal(decoded, f)
	}
	return json.Unmarshal([]byte(encoded), f)
}

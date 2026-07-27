package scout_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jaredtmartin/fido"
	"github.com/jaredtmartin/scout"
)

func handleGetDog(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Content(fido.String("Hello, World! Let's Get a Dog!"), fido.String("Here's some more content!"))
}
func handlePostDog(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Content(fido.String("Hello, World! Let's Post a Dog!"))
}
func handleDeleteDog(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Content(fido.String("Hello, World! Let's Delete a Dog!"))
}
func handlePutDog(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Content(fido.String("Hello, World! Let's Put a Dog!"))
}
func handlePatchDog(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Content(fido.String("Hello, World! Let's Patch a Dog!"))
}
func handleSimpleError(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Error(fmt.Errorf("Something went wrong!"))
}
func handleDetailedError(w http.ResponseWriter, r *http.Request) scout.Response {
	err := fmt.Errorf("Details about the error.")
	return scout.Error(fmt.Errorf("Something went wrong!: %w", err))
}
func handleRedirect(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Redirect("/redirected")
}
func handleFlashMessages(w http.ResponseWriter, r *http.Request) scout.Response {
	return scout.Content(fido.String("body content")).Success("success flash").Info("info flash").Warning("warning flash")
}

//	func handleErrorWithContent(w http.ResponseWriter, r *http.Request) Response {
//		return Response().Content(fido.Button("Log In").Href("/login")).Error("You're not logged in.")
//	}
type ExpectedResponse struct {
	Body           string
	Code           int
	Headers        map[string]string
	RequestHeaders map[string]string
}

func testRoute(server *httptest.Server, method, path string, expected ExpectedResponse, t *testing.T) {
	t.Helper()
	req, err := http.NewRequest(method, server.URL+path, nil)
	if err != nil {
		t.Fatalf("Failed to create %s request: %v", method, err)
	}
	for key, value := range expected.RequestHeaders {
		req.Header.Set(key, value)
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform %s request: %v", method, err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := buf.String()

	if body != expected.Body {
		t.Errorf("Expected response body %q for %s request, got %q", expected.Body, method, body)
	}
	if resp.StatusCode != expected.Code {
		t.Errorf("Expected response code %d for %s request, got %d", expected.Code, method, resp.StatusCode)
	}
	for key, value := range expected.Headers {
		if resp.Header.Get(key) != value {
			t.Errorf("Expected response header %s %q for %s request, got %q", key, value, method, resp.Header.Get(key))
		}
	}
}
func layout(w http.ResponseWriter, r *http.Request, elements ...fido.Element) fido.Element {
	return fido.NewElement("layout").Children(elements...)
}
func flashRenderer(flash scout.Flash, i int) fido.Element {
	return fido.Div("notification").Text(flash.Message).Class(string(flash.Urgency))
}
func errorPage(err scout.Response) fido.Element {
	// get everything before the : in the error message
	return fido.NewElement("div").Children(
		fido.NewElement("msg").Text(err.ErrPublic()),
		fido.NewElement("detail").Text(err.ErrDetail()),
	)
}

func TestGet(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/dog").Get(handleGetDog)
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/dog", ExpectedResponse{
		Body:    "<layout>Hello, World! Let's Get a Dog!Here's some more content!</layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "POST", "/dog", ExpectedResponse{
		Body:    "Method Not Allowed\n",
		Code:    http.StatusMethodNotAllowed,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "GET", "/cat", ExpectedResponse{
		Body:    "404 page not found\n",
		Code:    http.StatusNotFound,
		Headers: map[string]string{},
	}, t)
}
func TestMultiMethod(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/dog").
		Get(handleGetDog).
		Post(handlePostDog).
		Delete(handleDeleteDog).
		Put(handlePutDog).
		Patch(handlePatchDog)

	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/dog", ExpectedResponse{
		Body:    "<layout>Hello, World! Let's Get a Dog!Here's some more content!</layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "POST", "/dog", ExpectedResponse{
		Body:    "<layout>Hello, World! Let's Post a Dog!</layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "DELETE", "/dog", ExpectedResponse{
		Body:    "<layout>Hello, World! Let's Delete a Dog!</layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "PUT", "/dog", ExpectedResponse{
		Body:    "<layout>Hello, World! Let's Put a Dog!</layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "PATCH", "/dog", ExpectedResponse{
		Body:    "<layout>Hello, World! Let's Patch a Dog!</layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "PATCH", "/cat", ExpectedResponse{
		Body:    "404 page not found\n",
		Code:    http.StatusNotFound,
		Headers: map[string]string{},
	}, t)
}
func TestPost(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/dog").Post(handlePostDog)
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "POST", "/dog", ExpectedResponse{
		Body:    "<layout>Hello, World! Let's Post a Dog!</layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "GET", "/dog", ExpectedResponse{
		Body:    "Method Not Allowed\n",
		Code:    http.StatusMethodNotAllowed,
		Headers: map[string]string{},
	}, t)
	testRoute(server, "POST", "/cat", ExpectedResponse{
		Body:    "404 page not found\n",
		Code:    http.StatusNotFound,
		Headers: map[string]string{},
	}, t)
}
func TestErrors(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/err").Get(handleSimpleError)
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/err", ExpectedResponse{
		Body:    "<layout><div><msg>Something went wrong!</msg><detail></detail></div></layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
	router.Path("/err3").Get(handleDetailedError)
	testRoute(server, "GET", "/err3", ExpectedResponse{
		Body:    "<layout><div><msg>Something went wrong!</msg><detail>Details about the error.</detail></div></layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)

}
func TestStandardRedirect(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/redirect").Get(handleRedirect)
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/redirect", ExpectedResponse{
		Body: "<a href=\"/redirected\">See Other</a>.\n\n",
		Code: http.StatusSeeOther,
		Headers: map[string]string{
			"Location": "/redirected",
		},
	}, t)
}
func TestHtmxRedirect(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/redirect").Get(handleRedirect)
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/redirect", ExpectedResponse{
		Body: "",
		Code: http.StatusOK,
		Headers: map[string]string{
			"HX-Redirect": "/redirected",
		},
		RequestHeaders: map[string]string{
			"HX-Request": "true",
		},
	}, t)
}
func TestHtmxRedirectToSamePath(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/redirect").Get(handleRedirect)
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/redirect", ExpectedResponse{
		Body: "",
		Code: http.StatusOK,
		Headers: map[string]string{
			"HX-Redirect": "/redirected",
		},
		RequestHeaders: map[string]string{
			"HX-Request": "true",
		},
	}, t)
}

func TestFlashMessages(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, errorPage, flashRenderer)
	router.Path("/flash").Get(handleFlashMessages)
	server := httptest.NewServer(mux)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar}

	firstResp, err := client.Get(server.URL + "/flash")
	if err != nil {
		t.Fatalf("failed to perform first request: %v", err)
	}
	defer firstResp.Body.Close()

	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d on first request, got %d", http.StatusOK, firstResp.StatusCode)
	}

	secondResp, err := client.Get(server.URL + "/flash")
	if err != nil {
		t.Fatalf("failed to perform second request: %v", err)
	}
	defer secondResp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(secondResp.Body); err != nil {
		t.Fatalf("failed to read second response body: %v", err)
	}

	body := buf.String()
	for _, message := range []string{"success flash", "info flash", "warning flash"} {
		if !strings.Contains(body, message) {
			t.Fatalf("expected response body to contain %q, got %q", message, body)
		}
	}
	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d on second request, got %d", http.StatusOK, secondResp.StatusCode)
	}
}

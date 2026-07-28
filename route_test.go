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

// func errorPage(err scout.Response) fido.Element {
// 	// get everything before the : in the error message
// 	return fido.NewElement("div").Children(
// 		fido.NewElement("msg").Text(err.ErrPublic()),
// 		fido.NewElement("detail").Text(err.ErrDetail()),
// 	)
// }

func TestGet(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/dog").Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Content(fido.String("Hello, World! Let's Get a Dog!"), fido.String("Here's some more content!"))
	})
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
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/dog").
		Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
			return scout.Content(fido.String("Hello, World! Let's Get a Dog!"), fido.String("Here's some more content!"))
		}).
		Post(func(w http.ResponseWriter, r *http.Request) scout.Response {
			return scout.Content(fido.String("Hello, World! Let's Post a Dog!"))
		}).
		Delete(func(w http.ResponseWriter, r *http.Request) scout.Response {
			return scout.Content(fido.String("Hello, World! Let's Delete a Dog!"))
		}).
		Put(func(w http.ResponseWriter, r *http.Request) scout.Response {
			return scout.Content(fido.String("Hello, World! Let's Put a Dog!"))
		}).
		Patch(func(w http.ResponseWriter, r *http.Request) scout.Response {
			return scout.Content(fido.String("Hello, World! Let's Patch a Dog!"))
		})

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
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/dog").Post(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Content(fido.String("Hello, World! Let's Post a Dog!"))
	})
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
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/err").Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Error(fmt.Errorf("Something went wrong!"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	testRoute(server, "GET", "/err", ExpectedResponse{
		Body:    "<layout><div class=\"error notification\">Something went wrong!</div><div class=\"content\"></div></layout>",
		Code:    http.StatusInternalServerError,
		Headers: map[string]string{},
	}, t)
	// router.Path("/err3").Get(handleDetailedError)
	// testRoute(server, "GET", "/err3", ExpectedResponse{
	// 	Body:    "<layout><div><msg>Something went wrong!</msg><detail>Details about the error.</detail></div></layout>",
	// 	Code:    http.StatusOK,
	// 	Headers: map[string]string{},
	// }, t)

}
func TestSuccess(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/success").Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Success("Success!")
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/success", ExpectedResponse{
		Body:    "<layout><div class=\"notification success\">Success!</div><div class=\"content\"></div></layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
}
func TestWarning(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/warning").Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Respond().Warning("Warning message!")
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/warning", ExpectedResponse{
		Body:    "<layout><div class=\"notification warning\">Warning message!</div><div class=\"content\"></div></layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
}
func TestInfo(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/info").Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Respond().Info("Info message!")
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/info", ExpectedResponse{
		Body:    "<layout><div class=\"info notification\">Info message!</div><div class=\"content\"></div></layout>",
		Code:    http.StatusOK,
		Headers: map[string]string{},
	}, t)
}
func TestMultipleFlashes(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
	router.Path("/multiple").Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Respond().
			Success("First success!").
			Warning("A warning!").
			Info("Some info!").
			Error(fmt.Errorf("And an error!"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	testRoute(server, "GET", "/multiple", ExpectedResponse{
		Body:    "<layout><div class=\"notification success\">First success!</div><div class=\"notification warning\">A warning!</div><div class=\"info notification\">Some info!</div><div class=\"error notification\">And an error!</div><div class=\"content\"></div></layout>",
		Code:    http.StatusInternalServerError,
		Headers: map[string]string{},
	}, t)
}
func TestStandardRedirect(t *testing.T) {
	handleRedirect := func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Redirect("/redirected")
	}
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
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
	handleRedirect := func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Redirect("/redirected")
	}
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
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
	handleRedirect := func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Redirect("/redirected")
	}
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
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
	handleFlashMessages := func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Content(fido.String("body content")).Success("success flash").Info("info flash").Warning("warning flash")
	}
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)
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

func TestPostRedirectGet(t *testing.T) {
	mux := http.NewServeMux()
	router := scout.New(mux, layout, flashRenderer)

	// POST handler that redirects and sets a success flash
	router.Path("/submit").Post(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Redirect("/result").Success("Submission successful!")
	})

	// GET handler that renders the page
	router.Path("/result").Get(func(w http.ResponseWriter, r *http.Request) scout.Response {
		return scout.Respond()
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	// We use ErrUseLastResponse so we can manually inspect each stage of the redirect
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 1. Send POST request to /submit
	postResp, err := client.Post(server.URL+"/submit", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("failed to perform POST request: %v", err)
	}
	defer postResp.Body.Close()

	// Verify redirect status code (303 See Other)
	if postResp.StatusCode != http.StatusSeeOther {
		t.Errorf("expected status code %d on POST redirect, got %d", http.StatusSeeOther, postResp.StatusCode)
	}

	// Verify redirect Location header
	location := postResp.Header.Get("Location")
	if location != "/result" {
		t.Errorf("expected redirect location '/result', got %q", location)
	}

	// 2. Perform the GET request to the redirect destination /result
	getResp1, err := client.Get(server.URL + location)
	if err != nil {
		t.Fatalf("failed to perform first GET request: %v", err)
	}
	defer getResp1.Body.Close()

	if getResp1.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d on first GET request, got %d", http.StatusOK, getResp1.StatusCode)
	}

	buf1 := new(bytes.Buffer)
	if _, err := buf1.ReadFrom(getResp1.Body); err != nil {
		t.Fatalf("failed to read first GET response body: %v", err)
	}

	body1 := buf1.String()
	expectedFlashHtml := "<div class=\"notification success\">Submission successful!</div>"
	if !strings.Contains(body1, expectedFlashHtml) {
		t.Errorf("expected first GET response body to contain %q, got %q", expectedFlashHtml, body1)
	}

	// 3. Perform a second GET request to /result to ensure flash was consumed and is gone
	getResp2, err := client.Get(server.URL + location)
	if err != nil {
		t.Fatalf("failed to perform second GET request: %v", err)
	}
	defer getResp2.Body.Close()

	if getResp2.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d on second GET request, got %d", http.StatusOK, getResp2.StatusCode)
	}

	buf2 := new(bytes.Buffer)
	if _, err := buf2.ReadFrom(getResp2.Body); err != nil {
		t.Fatalf("failed to read second GET response body: %v", err)
	}

	body2 := buf2.String()
	if strings.Contains(body2, expectedFlashHtml) {
		t.Errorf("expected second GET response body to NOT contain flash %q, but it did: %q", expectedFlashHtml, body2)
	}
}

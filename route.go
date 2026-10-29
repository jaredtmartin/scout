package scout

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"sort"
	"strings"

	"github.com/jaredtmartin/fido"
)

type Layout func(w http.ResponseWriter, r *http.Request, flashes Flashes, content []fido.Element) fido.Element
type Handler func(http.ResponseWriter, *http.Request) Response
type BranchType map[string]*PathType
type ErrorPageType func(err Response) fido.Element

// GuardFunc decides whether a request may proceed to a Secured route's handler.
// Return nil to allow the request. Return a Response to deny it; the response is
// rendered exactly like a handler response (e.g. Redirect("/login") to send
// unauthenticated users to a login page).
type GuardFunc func(http.ResponseWriter, *http.Request) *Response

// PathType holds the method handlers for a single path along with that path's
// auth declaration. By default, every path requires authentication: unless a
// path explicitly opts in via Public(), auth is enforced via its Secured(guard)
// (or the router's DefaultGuard), and any undeclared path with no guard is
// denied with HTTP 403 Forbidden. This makes "forgot to secure the endpoint"
// impossible to ship silently.
type PathType struct {
	methods  map[string]Handler
	guard    GuardFunc
	public   bool
	declared bool
}

type Response struct {
	content  []fido.Element
	headers  map[string]string
	redirect string
	flashes  *Flashes
	code     int
}

func (r Response) Error(err error, expiry ...int) Response {
	r.code = http.StatusInternalServerError
	return r.Notify(err.Error(), ErrorUrgency, expiry...)
}

// NotFound sets the response's status code to HTTP 404 Not Found.
// Call Notify/Content/etc. on the returned response to add messages or content.
func (r Response) NotFound(msg string, expiry ...int) Response {
	r.code = http.StatusNotFound
	if msg != "" {
		return r.Notify(msg, ErrorUrgency, expiry...)
	}
	return r
}

// Forbidden sets the response's status code to HTTP 403 Forbidden.
func (r Response) Forbidden(msg string, expiry ...int) Response {
	r.code = http.StatusForbidden
	if msg != "" {
		return r.Notify(msg, ErrorUrgency, expiry...)
	}
	return r
}
func (r Response) Content(content ...fido.Element) Response {
	r.content = content
	r.code = http.StatusOK
	return r
}
func (r Response) Notify(msg string, urgency Urgency, expiry ...int) Response {
	var expiryInt int = defaultExpiry
	if len(expiry) > 0 {
		expiryInt = expiry[0]
	}
	r.flashes.Add(msg, urgency, expiryInt)
	return r
}
func (r Response) Success(msg string, expiry ...int) Response {
	return r.Notify(msg, SuccessUrgency, expiry...)
}
func (r Response) Info(msg string, expiry ...int) Response {
	return r.Notify(msg, InfoUrgency, expiry...)
}
func (r Response) Warning(msg string, expiry ...int) Response {
	return r.Notify(msg, WarningUrgency, expiry...)
}
func (r Response) Header(key, value string) Response {
	r.headers[key] = value
	return r
}
func (r Response) Redirect(url string) Response {
	r.redirect = url
	return r
}
func (r Response) Back() Response {
	r.headers["HX-Back"] = "true"
	return r
}
func (r Response) PushUrl(url string) Response {
	r.headers["HX-Push-Url"] = url
	return r
}
func (r Response) ReplaceUrl(url string) Response {
	r.headers["HX-Replace-Url"] = url
	return r
}

func (r Response) GetContent() []fido.Element {
	return r.content
}

func Respond() Response {
	return Response{
		headers: make(map[string]string),
		flashes: NewFlash(),
		content: []fido.Element{fido.Div("content")},
		code:    http.StatusOK,
	}
}
func Content(content ...fido.Element) Response {
	return Respond().Content(content...)
}
func Error(err error, expiry ...int) Response {
	res := Respond().Error(err)
	return res
}

// NotFound signals that the requested resource does not exist.
// The optional message is shown as an error-urgency flash notification.
func NotFound(msg string, expiry ...int) Response {
	return Respond().NotFound(msg, expiry...)
}
func Success(msg string, expiry ...int) Response {
	return Respond().Success(msg, expiry...)
}
func Warning(msg string, expiry ...int) Response {
	return Respond().Warning(msg, expiry...)
}
func Info(msg string, expiry ...int) Response {
	return Respond().Info(msg, expiry...)
}
func Redirect(url string) Response {
	return Respond().Redirect(url)
}
func Back() Response {
	return Respond().Back()
}

type Router struct {
	layout       Layout
	routes       BranchType
	errorPage    ErrorPageType
	Mux          *http.ServeMux
	verbose      bool
	defaultGuard GuardFunc
}

// DefaultGuard sets a router-wide guard that is enforced on every route that
// does not call Public() and does not provide its own Secured(guard).
func (router *Router) DefaultGuard(guard GuardFunc) *Router {
	router.defaultGuard = guard
	return router
}

// CheckAuth returns an error listing every path that registered handlers but
// never declared its access level via Public() or Secured() (when no router
// DefaultGuard is set).
func (router *Router) CheckAuth() error {
	if router.defaultGuard != nil {
		return nil
	}
	var undeclared []string
	for path, p := range router.routes {
		if p != nil && !p.declared {
			undeclared = append(undeclared, path)
		}
	}
	if len(undeclared) == 0 {
		return nil
	}
	sort.Strings(undeclared)
	return fmt.Errorf("%d route(s) with no auth declaration: %s", len(undeclared), strings.Join(undeclared, ", "))
}

func Branch() BranchType {
	return make(BranchType)
}
func New(mux *http.ServeMux, layout Layout) *Router {
	return &Router{
		layout: layout,
		routes: Branch(),
		Mux:    mux,
	}
}
func (router *Router) Handle(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	router.Mux.HandleFunc(pattern, handler)
}
func (router *Router) Verbose(verbose bool) {
	router.verbose = verbose
}
func (router *Router) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	router.Mux.HandleFunc(pattern, handler)
}
func (router *Router) Path(path string) *PathType {
	if router.routes[path] == nil {
		router.routes[path] = &PathType{methods: make(map[string]Handler)}
		router.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			pathHandler(w, r, router, router.routes[path])
		})
		if router.verbose {
			log.Println("Added route:", urlFromPath(path))
		}
	}
	// Re-registering the same path (e.g. two modules contributing methods)
	// preserves the existing auth declaration; Secured/Public overwrite it.
	return router.routes[path]
}

func (r *PathType) Handle(method string, handler Handler) *PathType {
	r.methods[method] = handler
	return r
}

func (r *PathType) Map(route PathType) *PathType {
	maps.Copy(r.methods, route.methods)
	return r
}

// Public marks this path as intentionally accessible without authentication.
// This is an affirmative declaration, not the absence of one.
func (r *PathType) Public() *PathType {
	r.declared = true
	r.public = true
	r.guard = nil
	return r
}

// Secured marks this path as authentication-protected and installs the guard
// that decides per-request access. The guard returns nil to allow, or a
// deny Response (e.g. a redirect to the login page).
func (r *PathType) Secured(guard GuardFunc) *PathType {
	if guard == nil {
		log.Panicf("Secured() called with a nil guard for path with handler(s) %v", r.methods)
	}
	r.declared = true
	r.public = false
	r.guard = guard
	return r
}

func (r *PathType) Get(handler Handler) *PathType {
	return r.Handle(http.MethodGet, handler)
}
func (r *PathType) Post(handler Handler) *PathType {
	return r.Handle(http.MethodPost, handler)
}
func (r *PathType) Delete(handler Handler) *PathType {
	return r.Handle(http.MethodDelete, handler)
}
func (r *PathType) Put(handler Handler) *PathType {
	return r.Handle(http.MethodPut, handler)
}
func (r *PathType) Patch(handler Handler) *PathType {
	return r.Handle(http.MethodPatch, handler)
}

// renderResponse applies a Response to the wire the same way handler responses
// are delivered (headers, redirects with flashes, or layout rendering).
func (router *Router) renderResponse(w http.ResponseWriter, r *http.Request, response Response) {
	for key, value := range response.headers {
		w.Header().Set(key, value)
	}
	redirect := response.redirect
	if redirect != "" {
		if response.flashes != nil && len(*response.flashes) > 0 {
			response.flashes.Save(w)
		}
		// If the request is not an HTMX request, we want to do a standard http redirect
		if r.Header.Get("HX-Request") == "" {
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}
		// If the request is an HTMX request
		// we send the special hx-redirect header
		// but only if the path is different from the current path
		if CurrentPath(r) != redirect {
			w.Header().Set("HX-Redirect", redirect)
		}
		return
	}
	// If not redirecting, load flashes from cookie, merge with new flashes, and render them
	cookieFlashes := loadFlashes(w, r)
	var allFlashes Flashes
	if cookieFlashes != nil {
		allFlashes = append(allFlashes, *cookieFlashes...)
	}
	if response.flashes != nil {
		allFlashes = append(allFlashes, *response.flashes...)
	}
	if response.code != http.StatusOK {
		w.WriteHeader(response.code)
	}
	router.layout(w, r, allFlashes, response.GetContent()).Send(w)
}

func pathHandler(w http.ResponseWriter, r *http.Request, router *Router, methods *PathType) {
	// Default-deny: unless a route explicitly opted into Public(), auth is enforced.
	if !methods.public {
		guard := methods.guard
		if guard == nil {
			guard = router.defaultGuard
		}
		if guard == nil {
			log.Printf("DENIED (undeclared route): %s %s — register with .Public() or .Secured(guard)", r.Method, r.URL.Path)
			router.renderResponse(w, r, Respond().Forbidden("This page is not configured for public access."))
			return
		}
		if deny := guard(w, r); deny != nil {
			router.renderResponse(w, r, *deny)
			return
		}
	}
	handler, ok := methods.methods[r.Method]
	if ok && handler != nil {
		response := handler(w, r)
		router.renderResponse(w, r, response)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

package scout

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"strings"

	"github.com/jaredtmartin/fido"
)

type Layout func(http.ResponseWriter, *http.Request, ...fido.Element) fido.Element
type Handler func(http.ResponseWriter, *http.Request) Response
type FlashRenderer func(flash Flash, i int) fido.Element
type PathType map[string]Handler
type BranchType map[string]*PathType
type ErrorPageType func(err Response) fido.Element

type Response struct {
	content  []fido.Element
	headers  map[string]string
	err      error
	redirect string
	flashes  *Flashes
}

func (r Response) Error(err error) Response {
	r.err = err
	return r
}
func (r Response) Content(content ...fido.Element) Response {
	r.content = content
	return r
}
func (r Response) Notify(msg string, urgency Urgency, expiry ...int) Response {
	var expiryInt int = 5000
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
func (r Response) Err() error {
	return r.err
}
func (r Response) ErrPublic() string {
	if r.err == nil {
		return ""
	}
	parts := strings.Split(r.err.Error(), ":")
	if len(parts) < 1 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
func (r Response) ErrDetail() string {
	if r.err == nil {
		return ""
	}
	parts := strings.Split(r.err.Error(), ":")
	if len(parts) < 2 {
		return ""
	}
	// join all the parts after the first one with :
	return strings.TrimSpace(strings.Join(parts[1:], ": "))
}
func (r Response) GetContent() []fido.Element {
	return r.content
}

// Error(err).WrapErr("This dog has wandered off.")
func (r Response) Wrap(msg string) Response {
	r.err = fmt.Errorf("%s: %w", msg, r.err)
	return r
}
func Respond() Response {
	return Response{
		headers: make(map[string]string),
		flashes: NewFlash(),
	}
}
func Content(content ...fido.Element) Response {
	return Respond().Content(content...)
}
func Error(err error) Response {
	res := Respond().Error(err)
	return res
}
func Success(msg string) Response {
	return Respond().Success(msg)
}
func Redirect(url string) Response {
	return Respond().Redirect(url)
}
func Back() Response {
	return Respond().Back()
}

type Router struct {
	layout        Layout
	routes        BranchType
	errorPage     ErrorPageType
	flashRenderer FlashRenderer
	Mux           *http.ServeMux
	verbose       bool
}

func Branch() BranchType {
	return make(BranchType)
}
func New(mux *http.ServeMux, layout Layout, errorPage ErrorPageType, flashRenderer FlashRenderer) *Router {
	return &Router{
		layout:        layout,
		routes:        Branch(),
		errorPage:     errorPage,
		flashRenderer: flashRenderer,
		Mux:           mux,
	}
}

//	func (router *Router) Route(router *Router) *Router {
//		for path, route := range routes {
//			router.Path(path).Map(*route)
//		}
//		return router
//	}
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
		router.routes[path] = &PathType{}
	}
	router.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		renderedFlashes := RenderFlashes(router.flashRenderer, w, r)
		pathHandler(w, r, router, *router.routes[path], renderedFlashes)
	})
	if router.verbose {
		log.Println("Added route:", urlFromPath(path))
	}
	return router.routes[path]
}

func (r *PathType) Handle(method string, handler Handler) *PathType {
	(*r)[method] = handler
	return r
}

func (r *PathType) Map(route PathType) *PathType {
	maps.Copy(*r, route)
	return r
}

func (r *PathType) Get(handler Handler) *PathType {
	(*r)[http.MethodGet] = handler
	return r
}
func (r *PathType) Post(handler Handler) *PathType {
	(*r)[http.MethodPost] = handler
	return r
}
func (r *PathType) Delete(handler Handler) *PathType {
	(*r)[http.MethodDelete] = handler
	return r
}
func (r *PathType) Put(handler Handler) *PathType {
	(*r)[http.MethodPut] = handler
	return r
}
func (r *PathType) Patch(handler Handler) *PathType {
	(*r)[http.MethodPatch] = handler
	return r
}

func pathHandler(w http.ResponseWriter, r *http.Request, router *Router, methods PathType, renderedFlashes fido.Element) {
	if handler, ok := (methods)[r.Method]; ok && handler != nil {
		response := handler(w, r)
		for key, value := range response.headers {
			w.Header().Set(key, value)
		}
		response.flashes.Save(w)
		redirect := response.redirect
		if redirect != "" {
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
		if response.Err() != nil {
			if r.Header.Get("HX-Request") != "" {
				http.Error(w, response.ErrPublic(), http.StatusInternalServerError)
				return
			}
			router.layout(w, r, router.errorPage(response)).Send(w)
			// renderedFlashes.Send(w)
			return
		}
		content := append([]fido.Element{renderedFlashes}, response.GetContent()...)
		router.layout(w, r, content...).Send(w)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

package scout

import (
	"log"
	"maps"
	"net/http"

	"github.com/jaredtmartin/fido"
)

type Layout func(w http.ResponseWriter, r *http.Request, flashes Flashes, content []fido.Element) fido.Element
type Handler func(http.ResponseWriter, *http.Request) Response
type PathType map[string]Handler
type BranchType map[string]*PathType
type ErrorPageType func(err Response) fido.Element

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
	layout    Layout
	routes    BranchType
	errorPage ErrorPageType
	Mux       *http.ServeMux
	verbose   bool
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
		pathHandler(w, r, router, *router.routes[path])
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

func pathHandler(w http.ResponseWriter, r *http.Request, router *Router, methods PathType) {
	if handler, ok := (methods)[r.Method]; ok && handler != nil {
		response := handler(w, r)
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
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

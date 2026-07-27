package scout

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func Path(root string, id string, suffixes ...string) string {
	parts := []string{root, id}
	parts = append(parts, suffixes...)
	parts = filterEmptyStrings(parts)
	return "/" + strings.Join(parts, "/")
}

func filterEmptyStrings(parts []string) []string {
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
func Url(root string, id string, suffixes ...string) string {
	return urlFromPath(Path(root, id, suffixes...))
}
func urlFromPath(path string) string {
	host := os.Getenv("HOST")
	env := os.Getenv("ENV")
	protocol := "http"
	if env == "production" {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s%s", protocol, host, path)
}
func CurrentPath(r *http.Request) string {
	u, err := url.Parse(r.Header.Get("HX-Current-URL"))
	if err == nil {
		return u.Path
	}
	return r.URL.Path
}

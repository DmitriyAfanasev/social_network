// Package swagger выдаёт единую документацию публичного API gateway.
package swagger

import (
	"embed"
	"net/http"

	gatewaydocs "general-project/gateway/docs"
)

//go:embed ui/index.html
var ui embed.FS

// Redirect перенаправляет короткий адрес Swagger на страницу документации.
func Redirect(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, "/swagger/", http.StatusMovedPermanently)
}

// UI возвращает страницу Swagger UI.
func UI(writer http.ResponseWriter, _ *http.Request) {
	content, err := ui.ReadFile("ui/index.html")
	if err != nil {
		http.Error(writer, "swagger ui is unavailable", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writer.Write(content)
}

// JSON возвращает единую OpenAPI-спецификацию в формате JSON.
func JSON(writer http.ResponseWriter, _ *http.Request) {
	serveSpec(writer, "swagger.json", "application/json; charset=utf-8")
}

// YAML возвращает единую OpenAPI-спецификацию в формате YAML.
func YAML(writer http.ResponseWriter, _ *http.Request) {
	serveSpec(writer, "swagger.yaml", "text/yaml; charset=utf-8")
}

func serveSpec(writer http.ResponseWriter, name string, contentType string) {
	content, err := gatewaydocs.Files.ReadFile(name)
	if err != nil {
		http.Error(writer, "swagger specification is unavailable", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", contentType)
	_, _ = writer.Write(content)
}

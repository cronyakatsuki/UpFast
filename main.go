package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	t := &Template{
		templates: template.Must(template.ParseGlob("public/views/*.html")),
	}

	e := echo.New()

	e.Renderer = t

	e.GET("/", Index)

	e.Logger.Fatal(e.Start(":1323"))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func Index(c echo.Context) error {
	return c.Render(http.StatusOK, "index", map[string]interface{}{
		"host":        c.Request().Host,
		"upload_only": true,
	})
}

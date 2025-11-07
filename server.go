package main

import (
	"embed"
	"flag"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data any, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

//go:embed all:public/views/*.html
var templates embed.FS

//go:embed all:static/*
var static embed.FS

// domain of the website
var domain string

func StartServer() {
	t := &Template{
		templates: template.Must(template.ParseFS(templates, "public/views/*.html")),
	}

	p := flag.Int("p", 1323, "upfast port to listen on.")
	a := flag.String("a", "127.0.0.1", "upfast ip to listen to")
	d := flag.String("d", "127.0.0.1", "upfast domain")
	flag.Parse()

	host := *a + ":" + strconv.Itoa(*p)

	domain = *d

	e := echo.New()

	e.Renderer = t
	e.Use(middleware.Logger())

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:       "static",
		Browse:     false,
		HTML5:      true,
		Filesystem: http.FS(static),
	}))

	files := "files"

	CreateDirIfNotExisting(files)

	e.Static("/files", files)

	e.GET("/", Index)

	e.POST("/", Upload)

	e.GET("/files/", Files)

	e.DELETE("/files/:file", Delete)

	e.Logger.Fatal(e.Start(host))
}

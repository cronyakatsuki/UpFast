package main

import (
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func main() {
	t := &Template{
		templates: template.Must(template.ParseGlob("public/views/*.html")),
	}

	e := echo.New()

	e.Renderer = t

	e.Static("/", "static")

	files := "files"

	if _, err := os.Stat(files); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(files, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}

	e.Static("/files", files)

	e.GET("/", Index)

	e.POST("/", Upload)

	e.GET("/files/", Files)

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

func Files(c echo.Context) error {
	return c.Render(http.StatusOK, "files", "test")
}

func Upload(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}

	dst, err := os.Create("files/" + file.Filename)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	fileUrl := c.Request().Host + "/files/" + file.Filename + "\n"

	return c.String(http.StatusOK, fileUrl)
}

package main

import (
	"embed"
	"errors"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed all:public/views/*.html
var templates embed.FS

//go:embed all:static/*
var static embed.FS

var domain string

func main() {
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

	e.DELETE("/files/:file", Delete)

	e.Logger.Fatal(e.Start(host))
}

type Template struct {
	templates *template.Template
}

type File struct {
	Name     string
	FileType string
	Content  string
}

type FilesData struct {
	Files []File
}

type IndexData struct {
	Host string
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func Index(c echo.Context) error {
	data := IndexData{
		Host: domain,
	}
	return c.Render(http.StatusOK, "index", data)
}

func GetFileContentType(ouput *os.File) (string, error) {
	buf := make([]byte, 512)

	_, err := ouput.Read(buf)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buf)

	return contentType, nil
}

func Files(c echo.Context) error {
	var files FilesData

	filelist, err := os.ReadDir("./files/")
	if err != nil {
		log.Fatal(err)
	}

	var Type string
	var Content string

	ImageMatch := regexp.MustCompile("^image/.*")
	VideoMatch := regexp.MustCompile("^video/.*")
	JsonMatch := regexp.MustCompile("application/json")
	TextMatch := regexp.MustCompile("^text/.*|application/octet-stream")

	for _, f := range filelist {
		filePath := "files/" + f.Name()
		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}

		defer file.Close()

		contentType, err := GetFileContentType(file)

		switch {
		case ImageMatch.MatchString(contentType):
			Type = "image"
			Content = ""
		case VideoMatch.MatchString(contentType):
			Type = "video"
			Content = ""
		case JsonMatch.MatchString(contentType):
			Type = "text"
			b, _ := os.ReadFile(filePath)
			Content = string(b)
		case TextMatch.MatchString(contentType):
			Type = "text"
			b, _ := os.ReadFile(filePath)
			Content = string(b)
		default:
			Type = "else"
			Content = ""
		}

		log.Print(contentType)
		files.Files = append(
			files.Files,
			File{Name: f.Name(), FileType: Type, Content: Content},
		)
	}

	return c.Render(http.StatusOK, "files", files)
}

func Upload(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}

	if _, err := os.Stat("files/" + file.Filename); err == nil {
		return c.String(http.StatusOK, "A file with the same name already exist's on the server.\n")
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

	fileUrl := domain + "/files/" + file.Filename + "\n"

	UserAgent := c.Request().UserAgent()

	log.Print(UserAgent)

	match, err := regexp.MatchString("^curl/.*", UserAgent)
	if err != nil {
		log.Fatal(err)
	}

	if match {
		return c.String(http.StatusOK, fileUrl)
	}

	return c.Redirect(http.StatusMovedPermanently, "/files/"+file.Filename)
}

func Delete(c echo.Context) error {
	file := c.Param("file")
	filePath := "files/" + file

	if _, err := os.Stat(filePath); err != nil {
		return c.String(http.StatusOK, "That file doesn't exist on the server\n")
	}

	err := os.Remove(filePath)
	if err != nil {
		return c.String(http.StatusOK, "Error while deleting "+file+"\n")
	}
	return c.String(http.StatusOK, "Deleted file from server\n")
}

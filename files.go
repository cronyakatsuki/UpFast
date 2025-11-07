package main

import (
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"os"
	"regexp"
)

type File struct {
	Name     string
	FileType string
	Content  string
}

type FilesData struct {
	Files []File
}

func Files(c echo.Context) error {
	var files FilesData

	filelist, err := os.ReadDir("./files/")
	if err != nil {
		log.Fatal(err)
	}

	UserAgent := c.Request().UserAgent()

	log.Print(UserAgent)

	match, err := regexp.MatchString("^curl/.*", UserAgent)
	if err != nil {
		log.Fatal(err)
	}

	if match {
		out := ""
		for _, f := range filelist {
			out += f.Name() + "\n"
		}
		return c.String(http.StatusOK, out)
	}

	var Type string
	var Content string

	ImageMatch := regexp.MustCompile("^image/.*")
	VideoMatch := regexp.MustCompile("^video/.*")
	JsonMatch := regexp.MustCompile("application/json")
	TextMatch := regexp.MustCompile("^text/.*")

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

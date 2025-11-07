package main

import (
	"github.com/labstack/echo/v4"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
)

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

	return c.HTML(http.StatusOK, "File uploaded at url: <strong>"+fileUrl+"</strong>")
}

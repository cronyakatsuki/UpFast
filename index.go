package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type IndexData struct {
	Host string
}

func Index(c echo.Context) error {
	data := IndexData{
		Host: domain,
	}
	return c.Render(http.StatusOK, "index", data)
}

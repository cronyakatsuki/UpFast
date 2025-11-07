package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
)

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
	return c.String(http.StatusOK, "Deleted file from server: "+file+"\n")
}

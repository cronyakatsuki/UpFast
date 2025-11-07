package main

import (
	"errors"
	"log"
	"net/http"
	"os"
)

// Miscelaneous functions

// Create directory if it doesn't exist
func CreateDirIfNotExisting(d string) {
	if _, err := os.Stat(d); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(d, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
}

// Get the file content type
func GetFileContentType(ouput *os.File) (string, error) {
	buf := make([]byte, 512)

	_, err := ouput.Read(buf)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buf)

	return contentType, nil
}

package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	directory, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	fs := http.FileServer(http.Dir(directory))
	http.Handle("/", fs)
	http.ListenAndServe(":80", nil)
}

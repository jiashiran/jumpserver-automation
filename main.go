package main

import (
	"fmt"
	"jumpserver-automation/ws"
	"os"
)

func main() {

	f, _ := os.Open("/usr/local/db/logs/")
	fileInfo, _ := f.Readdir(-1)
	for _, info := range fileInfo {
		fmt.Println(info.Name())
		os.Remove("/usr/local/db/logs/" + info.Name())
	}
	f.Close()
	service()
}

func service() {
	ws.Service()
}

package main

import (
	"fmt"
	"os"
)

func main() {
	xmlFile, err := os.Open("open.kml")

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully opened open.kml")

	defer xmlFile.Close()
}

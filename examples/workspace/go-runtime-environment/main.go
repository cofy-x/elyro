package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Printf("APP_MODE=%s FILE_ONLY=%s\n", os.Getenv("APP_MODE"), os.Getenv("FILE_ONLY"))
}

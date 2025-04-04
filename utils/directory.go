package utils

import (
	"fmt"
	"os"
)

func MissingDirectory(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Println("Error creating store directory:", err)
			return true
		}
	}
	return false
}

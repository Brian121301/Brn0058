package builtins

import (
	"fmt"
	"os"
)

// Simple function that takes a single string and create a directory in the current working directory with that name.
func Mkdir(directory string) error {
	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("New directory created:", directory)
	}
	return nil
}

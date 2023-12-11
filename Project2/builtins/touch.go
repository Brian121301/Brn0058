package builtins

import (
	"fmt"
	"os"
)

func Touch(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("File created/updated:", filename)
	}
	file.Close()
	return nil
}

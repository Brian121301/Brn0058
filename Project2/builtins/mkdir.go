package builtins

import (
	"fmt"
	"os"
)


func Mkdir(directory string) error {
	if err := os.Mkdir(directory, os.ModePerm); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("New directory created:", directory)
	}
	return nil
}

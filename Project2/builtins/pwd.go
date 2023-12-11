package builtins

import (
	"fmt"
	"os"
)

func Pwd() error {
	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		//error catching
		fmt.Println("Error:", err)
		return nil
	}
	fmt.Println(currentDir)
	return nil
}

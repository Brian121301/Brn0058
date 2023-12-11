package builtins

import (
	"fmt"
	"os"
)

func Pwd() error {

	currentDir, err := os.Getwd()
	if err != nil {

		fmt.Println("Error:", err)
		return nil
	}
	fmt.Println(currentDir)
	return nil
}

package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const cost = 10

func main() {
	if len(os.Args) != 2 {
		panic("need exactly one arg: password to hash")
	}

	pass := os.Args[1]
	hashed, err := bcrypt.GenerateFromPassword([]byte(pass), cost)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(hashed))
}

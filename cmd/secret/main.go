package main

import (
	"fmt"
	"laundry-api/helper"
	"log"
)

func main() {
	secret, err := helper.RandomToken()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(secret)
}

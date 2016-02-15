package main

import (
	"fmt"

	"github.com/sara-nl/docker-1.9.1/pkg/namesgenerator"
)

func main() {
	fmt.Println(namesgenerator.GetRandomName(0))
}

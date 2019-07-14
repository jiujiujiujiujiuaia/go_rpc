package main

import (
	"fmt"
	"github.com/vmihailenco/msgpack"
)

func main() {
	a := "123"
	b,_ := msgpack.Marshal(a)
	b = b
	fmt.Println(b)
}

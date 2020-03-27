package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
)

func main() {
	bf := bytes.NewBuffer(nil)
	bf.WriteString("hello world!")

	out, _ := ioutil.ReadAll(bf)
	fmt.Println(string(out))

}

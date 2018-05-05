package main

import (
	"fmt"
	"github.com/wetdesertrock/settings"
)

type B struct {
	Name string
}

type A struct {
	Value int
	Others []B
}

type APIConf struct {
	Baseurl string
	Timeout int
}

func main() {
	settings := NewSettings()
	fmt.Println(settings.LoadJSONFile("./test.json"))
	fmt.Println(settings.LoadJSONFile("./test2.json"))
	settings.Print()
	fmt.Println(settings.GetString("three", "nonono"))
	fmt.Println(settings.GetFloat("four", -1))
	fmt.Println(settings.GetInt("five:six", -1))
	fmt.Println(settings.GetString("five:eight", "nonono"))
	fmt.Println(settings.GetString("seven", "nonono"))

	umtest := A{}
	err := settings.Get("unmarshal:test", &umtest)
	fmt.Println(err)
	fmt.Println(umtest)
}

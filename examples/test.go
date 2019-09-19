package main

import (
	"fmt"

	"github.com/wetdesertrock/flexiconfig"
)

type B struct {
	Name string
}

type A struct {
	Value  int
	Others []B
}

type APIConf struct {
	Baseurl string
	Timeout int
}

func main() {
	settings := flexiconfig.NewSettings()
	fmt.Println(settings.LoadJSONFile("./test.json"))
	fmt.Println(settings.LoadLuaFile("./test2.lua"))
	// 	fmt.Println(settings.LoadLuaString(`return {
	// 		three = "333",
	// 		seven = "7",
	// 		five = {
	// 			eight = "8"
	// 		}
	// 	}
	// `))
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

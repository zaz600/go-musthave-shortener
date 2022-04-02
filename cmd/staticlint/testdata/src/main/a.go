package main

import "os"

func main() {
	os.Exit(-1) // want `don't use os.Exit\(\) in main`
}

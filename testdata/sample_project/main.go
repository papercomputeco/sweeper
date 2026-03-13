package main

import "fmt"

func main() {
	result := doWork()
	if result != nil {
		fmt.Println(result)
	}
	fmt.Println("done")
}

func doWork() error {
	return nil
}

// Handle exported without proper doc comment style
func Handle(x int) {
	if x > 0 {
		fmt.Println(x)
	} else {
		if x < -10 {
			fmt.Println("very negative")
		}
	}
}

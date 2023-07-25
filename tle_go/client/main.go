package main

import "fmt"

func main() {
	fmt.Println("Please input which program you want to run:")
	fmt.Println("1: TLE-GRPC-Client, 2: TLE-Block-Listener")
	var i int
	fmt.Scan(&i)
	if i == 1 {
		fmt.Println("Running TLE-GRPC-Client")
		client_main()
	} else if i == 2 {
		fmt.Println("Running TLE-Block-Listener")
		listener_main()
	} else {
		fmt.Println("Invalid Input!!")
	}
}

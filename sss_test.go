package sss

import (
	"fmt"
	"runtime"
)

func Example3() {
	secret := "well hello there!" // our secret
	n := byte(30)                 // create 30 shares
	k := byte(2)                  // require 2 of them to combine

	shares, err := Split(n, k, []byte(secret)) // split into 30 shares
	if err != nil {
		fmt.Println(err)
		return
	}

	// select a random subset of the total shares
	subset := make(map[byte][]byte, k)
	for x, y := range shares { // just iterate since maps are randomized
		subset[x] = y
		if len(subset) == int(k) {
			break
		}
	}

	// combine two shares and recover the secret
	recovered := string(Combine(subset))
	fmt.Println(recovered)

	// Output: well hello there!
}

func Example() {
	secret := "well hello there!" // our secret
	n := byte(30)                 // create 30 shares
	k := byte(2)                  // require 2 of them to combine

	cpus := len(secret) / (runtime.NumCPU() + 10)

	send := make([]chan Input, cpus)
	ret := make(chan Result)
	quit := make([]chan bool, cpus)

	for i := 0; i < len(send); i++ {
		go SplitParallelLoop(send[i], ret, quit[i])
	}

	_, err := SplitParallel(n, k, []byte(secret), send, ret, cpus) // split into 30 shares
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < len(send); i++ {
		quit[i] <- true
	}
}

/*
func Example2() {
	secret := "well hello there!" // our secret
	n := byte(30)                 // create 30 shares
	k := byte(5)                  // require 3 of them to combine

	shares, err := SplitParallel(n, k, []byte(secret)) // split into 30 shares
	if err != nil {
		fmt.Println(err)
		return
	}

	// select a random subset of the total shares
	subset := make(map[byte][]byte, k)
	for x, y := range shares { // just iterate since maps are randomized
		subset[x] = y
		if len(subset) == int(k) {
			break
		}
	}

	// combine two shares and recover the secret
	recovered := string(CombineParallel(subset))
	fmt.Println(recovered)

	// Output: well hello there!
}
*/

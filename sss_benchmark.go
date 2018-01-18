package sss

import (
	"fmt"
	"time"
)

func Normal1() {
	secret := "The quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dog"
	n := byte(30) // create 30 shares
	k := byte(2)  // require 2 of them to combine

	start := time.Now()
	_, err := SplitParallel(n, k, []byte(secret)) // split into 30 shares
	t := time.Now()
	fmt.Println("Elapsed:", t.Sub(start))
	if err != nil {
		fmt.Println(err)
		return
	}
}

func Concur1() {
	secret := "The quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dog"
	n := byte(30) // create 30 shares
	k := byte(2)  // require 2 of them to combine

	start := time.Now()
	_, err := Split(n, k, []byte(secret)) // split into 30 shares
	t := time.Now()
	fmt.Println("Elapsed:", t.Sub(start))
	if err != nil {
		fmt.Println(err)
		return
	}
}

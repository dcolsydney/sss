package sss

import (
	"fmt"
	"runtime"
	"testing"
)

func BenchmarkNormal1(b *testing.B) {
	secret := "The quick brown fox jumped over the lazy dog"
	n := byte(30) // create 30 shares
	k := byte(3)  // require 2 of them to combine

	for i := 0; i < b.N; i++ {
		_, err := Split(n, k, []byte(secret)) // split into 30 shares
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func BenchmarkConcur1(b *testing.B) {
	secret := "The quick brown fox jumped over the lazy dog"
	n := byte(30) // create 30 shares
	k := byte(3)  // require 2 of them to combine

	length := len(secret)/(runtime.NumCPU()+10) + 1
	cpus := runtime.NumCPU() + 10

	send := make([]chan Input, length)
	for i := 0; i < len(send); i++ {
		send[i] = make(chan Input, 1000)
	}
	ret := make(chan Result)
	quit := make([]chan bool, length)
	for i := 0; i < len(send); i++ {
		quit[i] = make(chan bool, 1000)
	}

	for i := 0; i < len(send); i++ {
		go SplitParallelLoop(send[i], ret, quit[i])
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := SplitParallel(n, k, []byte(secret), send, ret, cpus) // split into 30 shares
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	for i := 0; i < len(send); i++ {
		quit[i] <- true
	}
}

func BenchmarkNormal2(b *testing.B) {
	secret := "well hello there!well hello there!well hello there!well hello there!"
	n := byte(20)
	k := byte(15)

	shares, err := Split(n, k, []byte(secret))
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

	b.ResetTimer()

	// combine two shares and recover the secret
	for i := 0; i < b.N; i++ {
		Combine(subset)
	}

	// Output: well hello there!
}

func BenchmarkConcur2(b *testing.B) {
	secret := "well hello there!well hello there!well hello there!well hello there!"
	n := byte(20)
	k := byte(15)

	shares, err := Split(n, k, []byte(secret))
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

	b.ResetTimer()

	// combine two shares and recover the secret
	for i := 0; i < b.N; i++ {
		CombineParallel(subset)
	}

}

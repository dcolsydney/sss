package sss

import (
	"fmt"
	"runtime"
	"testing"
)

var secret = "The quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dogThe quick brown fox jumped over the lazy dog"
var n = byte(30) // create 30 shares
var k = byte(3)  // require 2 of them to combine


func BenchmarkNormal1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Split(n, k, []byte(secret)) // split into 30 shares
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func BenchmarkNormalNew1(b *testing.B) {
	res, _ := SplitNew(n, k, []byte(secret))
	if string(Combine(res)) != secret {
		panic("Bad combine")
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := SplitNew(n, k, []byte(secret)) // split into 30 shares
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func BenchmarkConcurNew1(b *testing.B) {
	retChan := make([]chan []byte, n)
	sendChan := make([]chan Comp, n)
	endChan := make(chan bool, n)
	for i := byte(0); i < n; i++ {
		sendChan[i] = make(chan Comp)
		retChan[i] = make(chan []byte)
		go RunParallel(sendChan[i], retChan[i], endChan)
	}
	
	
	res, _ := SplitNewConcur(n, k, []byte(secret), sendChan, retChan)
	if string(Combine(res)) != secret {
		fmt.Println(string(Combine(res)), "!!!!!!!!!!", string(secret))
		panic("Bad combine")
	}
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := SplitNewConcur(n, k, []byte(secret), sendChan, retChan) // split into 30 shares
		if err != nil {
			panic(err)
		}
	}

	for i := byte(0); i < n; i++ {
		endChan <- true
	}
}

func BenchmarkConcur1(b *testing.B) {
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

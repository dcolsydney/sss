// Package sss implements Shamir's Secret Sharing algorithm over GF(2^8).
//
// Shamir's Secret Sharing algorithm allows you to securely share a secret with
// N people, allowing the recovery of that secret if K of those people combine
// their shares.
//
// It begins by encoding a secret as a number (e.g., 42), and generating a
// random polynomial equation of degree K-1 which has an X-intercept equal to
// the secret. Given K=3, the following equations might be generated:
//
//     f1(x) =  78x^2 +  19x + 42
//     f2(x) = 128x^2 + 171x + 42
//     f3(x) = 121x^2 +   3x + 42
//     f4(x) =  91x^2 +  95x + 42
//     etc.
//
// The polynomial is then evaluated for values 0 < X < N:
//
//     f1(1) =  139
//     f1(2) =  896
//     f1(3) = 1140
//     f1(4) = 1783
//     etc.
//
// These (x, y) pairs are the shares given to the parties. In order to combine
// shares to recover the secret, these (x, y) pairs are used as the input points
// for Lagrange interpolation, which produces a polynomial which matches the
// given points. This polynomial can be evaluated for f(0), producing the secret
// value--the common x-intercept for all the generated polynomials.
//
// If fewer than K shares are combined, the interpolated polynomial will be
// wrong, and the result of f(0) will not be the secret.
//
// This package constructs polynomials over the field GF(2^8) for each byte of
// the secret, allowing for fast splitting and combining of anything which can
// be encoded as bytes.
//
// This package has not been audited by cryptography or security professionals.
package sss

import (
	"crypto/rand"
	"errors"
	"fmt"
	"runtime"
)

var (
	// ErrInvalidCount is returned when the count parameter is invalid.
	ErrInvalidCount = errors.New("N must be >= K")
	// ErrInvalidThreshold is returned when the threshold parameter is invalid.
	ErrInvalidThreshold = errors.New("K must be > 1")
)

// Split the given secret into N shares of which K are required to recover the
// secret. Returns a map of share IDs (1-255) to shares.
func Split(n, k byte, secret []byte) (map[byte][]byte, error) {
	if k <= 1 {
		return nil, ErrInvalidThreshold
	}

	if n < k {
		return nil, ErrInvalidCount
	}

	shares := make(map[byte][]byte, n)

	for _, b := range secret {
		p, err := generate(k-1, b, rand.Reader)
		if err != nil {
			return nil, err
		}

		for x := byte(1); x <= n; x++ {
			shares[x] = append(shares[x], eval(p, x))
		}
	}

	return shares, nil
}


func SplitNew(n, k byte, secret []byte) (map[byte][]byte, error) {
	if k <= 1 {
		return nil, ErrInvalidThreshold
	}

	if n < k {
		return nil, ErrInvalidCount
	}

	shares := make(map[byte][]byte, n)

	p, err := generateRand(k, secret, rand.Reader)
	if err != nil {
		return nil, err
	}	
	// for i := 0; i < len(secret); i++ {
	// 	for x := byte(1); x <= n; x++ {
	// 		next := (i*int(k))
	// 		shares[x] = append(shares[x], eval(p[next:next+int(k)], x))
	// 	}
	// }

	for x := byte(1); x <= n; x++ {
		shares[x] = Compute(len(secret), k, x, p)
	}

	return shares, nil
}

type Comp struct {
	lenSecret int
	k byte
	x byte
	p []byte
}

func RunParallel(sendChan chan Comp, retChan chan []byte, endChan chan bool) {
	for {
		select {
		case m := <-sendChan:
			retChan <- Compute(m.lenSecret, m.k, m.x, m.p)
		case <-endChan:
			return
		}
	}
}		

func SplitNewConcur(n, k byte, secret []byte, sendChan []chan Comp, resultChan []chan []byte) (map[byte][]byte, error) {
	if k <= 1 {
		return nil, ErrInvalidThreshold
	}

	if n < k {
		return nil, ErrInvalidCount
	}

	shares := make(map[byte][]byte, n)

	p, err := generateRand(k, secret, rand.Reader)
	if err != nil {
		return nil, err
	}
	
	// for i := 0; i < len(secret); i++ {
	// 	for x := byte(1); x <= n; x++ {
	// 		next := (i*int(k))
	// 		shares[x] = append(shares[x], eval(p[next:next+int(k)], x))
	// 	}
	// }

	for x := byte(1); x <= n; x++ {
		sendChan[x-1] <- Comp{lenSecret: len(secret), k: k, x: x, p: p}
	}
	for x := byte(1); x <= n; x++ {
		shares[x] = <-resultChan[x-1]
	}

	return shares, nil
}

func Compute(lenSecret int, k byte, nIndex byte, p []byte) []byte {
	share := make([]byte, lenSecret)
	for i := 0; i < lenSecret; i++ {
		next := (i*int(k))
		share[i] = eval(p[next:next+int(k)], nIndex)
	}
	return share
}


type Result struct {
	Shares [][]byte
	Index  int
	N      int
}

type Input struct {
	Polys      [][]byte
	Secrets    []byte
	N          byte
	Start, End int
	Ret        chan Result
}

func SplitParallel(n, k byte, secret []byte, send []chan Input, ret chan Result, cpus int) (map[byte][]byte, error) {
	if k <= 1 {
		return nil, ErrInvalidThreshold
	}

	if n < k {
		return nil, ErrInvalidCount
	}

	p, err := generatePolys(k-1, secret, rand.Reader)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	count := 0
	for i := 0; i < len(secret); i += cpus {
		if i+cpus >= len(secret) {
			send[count] <- Input{Polys: p[i:], Secrets: secret[i:], N: n, Start: i, End: len(secret) - 1, Ret: ret}
			//		go SplitParallelLoop(p[i:], secret[i:], n, i, len(secret)-1, ret)
		} else {
			send[count] <- Input{Polys: p[i : i+cpus], Secrets: secret[i : i+cpus], N: n, Start: i, End: i + cpus - 1, Ret: ret}
			//		go SplitParallelLoop(p[i:i+cpus], secret[i:i+cpus], n, i, i+cpus-1, ret)
		}
		count++
	}

	shares := make(map[byte][]byte, n)
	for i := byte(1); i <= n; i++ {
		shares[i] = make([]byte, len(secret))
	}

	for count > 0 {
		count--
		res := <-ret
		for j := byte(1); j <= n; j++ {
			for i := 0; i < res.N; i++ {
				shares[j][i+res.Index] = res.Shares[i][int(j)-1]
			}
		}
	}

	return shares, nil
}

//func SplitParallelLoop(p [][]byte, bytes []byte, n byte, start_i, end_i int, ret chan Result) error {
func SplitParallelLoop(send chan Input, ret chan Result, quit chan bool) {
	for {
		select {
		case m := <-send:
			shares := make([][]byte, len(m.Secrets))
			for i := 0; i < len(m.Secrets); i++ {

				shares[i] = make([]byte, m.N)
				for x := byte(1); x <= m.N; x++ {
					shares[i][int(x)-1] = eval(m.Polys[i], x)
				}
			}

			res := Result{Shares: shares, Index: m.Start, N: len(m.Secrets)}
			//	res.Init(shares, start_i, len(bytes))
			ret <- res
		case <-quit:
			return
		}
	}
}

func (res *Result) Init(shares [][]byte, index int, n int) {
	res.Shares = make([][]byte, n)
	for i := 0; i < len(shares); i++ {
		res.Shares[i] = make([]byte, len(shares[i]))
		for j := 0; j < len(shares[i]); j++ {
			res.Shares[i][j] = shares[i][j]
		}
	}
	res.Index = index
	res.N = n
}

// Combine the given shares into the original secret.
//
// N.B.: There is no way to know whether the returned value is, in fact, the
// original secret.
func Combine(shares map[byte][]byte) []byte {
	var secret []byte
	for _, v := range shares {
		secret = make([]byte, len(v))
		break
	}

	points := make([]pair, len(shares))
	for i := range secret {
		p := 0
		for k, v := range shares {
			points[p] = pair{x: k, y: v[i]}
			p++
		}
		secret[i] = interpolate(points, 0)
	}

	return secret
}

func CombineParallel(shares map[byte][]byte) []byte {
	var secret []byte
	secret = make([]byte, len(shares))
	newShares := make([][]byte, len(shares))
	indices := make([]int, len(shares))

	c := 0
	for k, v := range shares {
		newShares[c] = v
		indices[c] = int(k)
		c++
	}

	for _, v := range shares {
		secret = make([]byte, len(v))
		break
	}

	cpus := runtime.NumCPU() + 10

	ret := make(chan Data)

	count := 0

	for i := 0; i < len(secret); i += cpus {
		var share [][]byte
		if i+cpus >= len(secret) {
			share = make([][]byte, len(secret)-i)
		} else {
			share = make([][]byte, cpus)
		}
		for j := 0; j < len(share); j++ {
			share[j] = make([]byte, len(newShares))
			for k := 0; k < len(newShares); k++ {
				share[j][k] = newShares[k][i+j]
			}
		}
		go CombineConcur(share, i, indices, ret)
		count++
	}

	/*
		for i := 0; i < len(secret); i++ {
			share := make([][]byte, 1)
			share[0] = make([]byte, 0)
			for j := 0; j < len(newShares); j++ {
				share[0] = append(share[0], newShares[j][i])
			}
			go CombineConcur(share, i, indices, ret)
			count++
		}
	*/

	for count > 0 {
		count--
		res := <-ret
		for i := 0; i < len(res.Secret); i++ {
			secret[i+res.Index] = res.Secret[i]
		}
	}
	return secret

}

type Data struct {
	Secret []byte
	Index  int
}

func CombineConcur(shares [][]byte, index int, indices []int, ret chan Data) {
	secret := make([]byte, len(shares))
	for i := range secret {
		points := make([]pair, len(shares[i]))
		for j := 0; j < len(shares[i]); j++ {
			points[j] = pair{x: byte(indices[j]), y: shares[i][j]}
		}
		secret[i] = interpolate(points, 0)
	}
	ret <- Data{Secret: secret, Index: index}
}

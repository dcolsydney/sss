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
	//"fmt"
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

type Result struct {
	Shares [][]byte
	Index  int
	N      int
}

func SplitParallel(n, k byte, secret []byte) (map[byte][]byte, error) {
	if k <= 1 {
		return nil, ErrInvalidThreshold
	}

	if n < k {
		return nil, ErrInvalidCount
	}

	cpus := runtime.NumCPU() + 10

	ret := make(chan Result)

	count := 0
	for i := 0; i < len(secret); i += cpus {
		if i+cpus >= len(secret) {
			go SplitParallelLoop(k-1, secret[i:], n, i, len(secret)-1, ret)
		} else {
			go SplitParallelLoop(k-1, secret[i:i+cpus], n, i, i+cpus-1, ret)
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

func SplitParallelLoop(k_1 byte, bytes []byte, n byte, start_i, end_i int, ret chan Result) error {
	shares := make([][]byte, len(bytes))
	for i := 0; i < len(bytes); i++ {
		p, err := generate(k_1, bytes[i], rand.Reader)
		if err != nil {
			return err
		}

		shares[i] = make([]byte, n)
		for x := byte(1); x <= n; x++ {
			shares[i][int(x)-1] = eval(p, x)
		}
	}

	res := Result{Shares: shares, Index: start_i, N: len(bytes)}
	//	res.Init(shares, start_i, len(bytes))
	ret <- res

	return nil
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

	cpus := runtime.NumCPU() + 3

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

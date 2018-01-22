package sss

import (
	//"fmt"
	"io"
)

// the degree of the polynomial
func degree(p []byte) int {
	return len(p) - 1
}

// evaluate the polynomial at the given point
func eval(p []byte, x byte) (result byte) {
	// Horner's scheme
	// tmp := result
	// for i := 0; i <= 1000; i++ {
	// 	result = mul(result, x) ^ p[len(p)-1] + byte(i)
	// }
	// result = tmp
	for i := 1; i <= len(p); i++ {
		result = mul(result, x) ^ p[len(p)-i]
	}
	return
}

func generateRand(k byte, secret []byte, ran io.Reader) ([]byte, error) {
	result := make([]byte, int(k)*len(secret))
	degree := k - 1
	if _, err := io.ReadFull(ran, result); err != nil {
		return nil, err
	}

	start := 0
	for _, b := range secret {
		result[start] = b
		nextDegree := start + int(degree)
		for {
			if result[nextDegree] != 0 {
				break
			}
			if _, err := io.ReadFull(ran, result[nextDegree:nextDegree+1]); err != nil {
				return nil, err
			}
		}
		start += int(k)
	}
	return result, nil
}



// generates a random n-degree polynomial w/ a given x-intercept
func generate(degree byte, x byte, ran io.Reader) ([]byte, error) {
	result := make([]byte, degree+1)
	result[0] = x

	buf := make([]byte, degree-1)
	if _, err := io.ReadFull(ran, buf); err != nil {
		return nil, err
	}

	for i := byte(1); i < degree; i++ {
		result[i] = buf[i-1]
	}

	// the Nth term can't be zero, or else it's a (N-1) degree polynomial
	for {
		buf = make([]byte, 1)
		if _, err := io.ReadFull(ran, buf); err != nil {
			return nil, err
		}

		if buf[0] != 0 {
			result[degree] = buf[0]
			return result, nil
		}
	}
}

func generatePolys(degree byte, x []byte, ran io.Reader) ([][]byte, error) {
	results := make([][]byte, len(x))
	for i := byte(0); i < byte(len(x)); i++ {
		results[i] = make([]byte, degree+1)
		results[i][0] = x[i]
	}
	//fmt.Println(results)

	buf := make([]byte, int(degree)*len(x))
	if _, err := io.ReadFull(ran, buf); err != nil {
		return nil, err
	}

	for i := 0; i < len(x); i++ {
		for j := byte(1); j <= degree; j++ {
			results[i][j] = buf[byte(i)*(degree)+j-1]
		}
	}

	//fmt.Println(results)

	for i := 0; i < len(x); i++ {
		buf = make([]byte, 1)
		if _, err := io.ReadFull(ran, buf); err != nil {
			i--
			continue
		}

		if buf[0] != 0 {
			results[i][degree] = buf[0]
		} else {
			i--
		}
	}
	return results, nil
}

// an input/output pair
type pair struct {
	x, y byte
}

// Lagrange interpolation
func interpolate(points []pair, x byte) (value byte) {
	for i, a := range points {
		weight := byte(1)
		for j, b := range points {
			if i != j {
				top := x ^ b.x
				bottom := a.x ^ b.x
				factor := div(top, bottom)
				weight = mul(weight, factor)
			}
		}
		value = value ^ mul(weight, a.y)
	}
	return
}

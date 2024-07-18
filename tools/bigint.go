package tools

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
)

func ParseBigInt(s string, base int) *big.Int {
	n := new(big.Int)
	n, ok := n.SetString(s, base)
	if !ok {
		fmt.Println("ParseBigInt: SetString error")
		return big.NewInt(0)
	}
	return n
}

func ConvertTOFloat64(f *big.Float) float64 {
	d, _ := f.Float64()
	return d
}

func BigIntToFloat64(b *big.Int, decimals uint64) float64 {
	dec := new(big.Int).SetUint64(decimals)
	f := new(big.Float).Quo(new(big.Float).SetInt(b), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), dec, nil)))
	return ConvertTOFloat64(f)
}

func Float64ToBigInt(b float64, decimals uint64) *big.Int {
	dec := new(big.Int).SetUint64(decimals)
	f := new(big.Float).Mul(new(big.Float).SetFloat64(b), new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), dec, nil)))
	return ToBigInt(f)
}

func PreservePrecision(f float64, length uint64) float64 {
	fmtStr := "%." + fmt.Sprintf("%d", length) + "f"
	a, _ := strconv.ParseFloat(fmt.Sprintf(fmtStr, f), 64)
	return a
}

func PowBigFloat(x, y *big.Float) *big.Float {
	// Convert x to a float64
	xf, _ := x.Float64()
	// Convert y to a float64
	yf, _ := y.Float64()

	// Perform the power operation using math.Pow
	result := math.Pow(xf, yf)

	// Convert the result back to a big.Float
	return big.NewFloat(result)
}

func ToBigInt(x *big.Float) *big.Int {
	i := new(big.Int)
	x.Int(i)
	return i
}

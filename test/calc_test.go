package main

import (
	"fmt"
	"math"
	"testing"

	"gonum.org/v1/gonum/optimize"
)

// 初始参数
var (
	L_A float64 = 1470.6128357485022 // 交易对A的ETH数量
	U_A float64 = 5241368.443797308  // 交易对A的USDT数量
	L_B float64 = 16.860177022056686 // 交易对B的ETH数量
	U_B float64 = 58887.3323355873   // 交易对B的USDT数量

	P_A float64 = U_A / L_A
	P_B float64 = U_B / L_B
)

// 目标函数：两个池交易后的价格相等
func equations(Q float64) float64 {
	P_A_after := (U_A - P_A*Q) / (L_A + Q)
	P_B_after := (U_B + P_B*Q) / (L_B - Q)
	return math.Abs(P_A_after - P_B_after)
}

// go test -v -run ^TestCalc$ github.com/xiangxn/listener/test
func TestCalc(t *testing.T) {
	// 初始猜测值
	Q_initial_guess := 1.0

	// 使用 optimize 包进行求解
	problem := optimize.Problem{
		Func: func(Q []float64) float64 {
			return equations(Q[0])
		},
	}

	result, err := optimize.Minimize(problem, []float64{Q_initial_guess}, nil, nil)
	if err != nil {
		fmt.Println("Optimization failed:", err)
		return
	}

	Q_optimal := result.X[0]

	// 计算交易后的价格
	P_A_after := (U_A - P_A*Q_optimal) / (L_A + Q_optimal)
	P_B_after := (U_B + P_B*Q_optimal) / (L_B - Q_optimal)

	fmt.Printf("Optimal Q: %f\n", Q_optimal)
	fmt.Printf("Price after arbitrage: %f (A), %f (B)\n", P_A_after, P_B_after)
}

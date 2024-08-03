package main

import (
	"fmt"
	"math"
	"testing"
	"time"
)

type Pair struct {
	From  string
	To    string
	Price float64
	Fee   float64
}

// go test -v -run ^TestTriangles$ github.com/xiangxn/listener/test
func TestTriangles(t *testing.T) {
	// 示例数据
	pairs := []Pair{
		{"A", "B", 2.0, 0.001},
		{"B", "C", 1.5, 0.001},
		{"C", "A", 0.34, 0.001},
		{"A", "C", 1.7, 0.001},
		{"C", "B", 0.65, 0.001},
		{"B", "A", 0.5, 0.001},
	}

	tt := time.Now()
	// 构建价格图
	graph := make(map[string]map[string]float64)
	for _, pair := range pairs {
		if graph[pair.From] == nil {
			graph[pair.From] = make(map[string]float64)
		}
		if graph[pair.To] == nil {
			graph[pair.To] = make(map[string]float64)
		}

		adjustedPrice := pair.Price * (1 - pair.Fee)
		graph[pair.From][pair.To] = -math.Log(adjustedPrice)
		graph[pair.To][pair.From] = math.Log(1 / adjustedPrice)

		// graph[pair.From][pair.To] = -math.Log(pair.Price)
		// graph[pair.To][pair.From] = math.Log(pair.Price)
	}

	// 寻找三角形路径
	triangles := [][]string{}
	for start := range graph {
		for mid := range graph[start] {
			if mid == start {
				continue
			}
			for end := range graph[mid] {
				if end == start || end == mid {
					continue
				}
				if _, exists := graph[end][start]; exists {
					triangles = append(triangles, []string{start, mid, end})
				}
			}
		}
	}

	// 计算潜在套利
	arbitrageOpportunities := []struct {
		Triangle []string
		Profit   float64
	}{}
	for _, triangle := range triangles {
		weight := graph[triangle[0]][triangle[1]] + graph[triangle[1]][triangle[2]] + graph[triangle[2]][triangle[0]]
		if weight < 0 {
			profit := math.Exp(-weight) - 1
			arbitrageOpportunities = append(arbitrageOpportunities, struct {
				Triangle []string
				Profit   float64
			}{Triangle: triangle, Profit: profit})
		}
	}

	fmt.Println("arbitrageOpportunities:", len(arbitrageOpportunities), " ", time.Since(tt))

	// 输出潜在套利路径
	for _, opportunity := range arbitrageOpportunities {
		fmt.Printf("Arbitrage opportunity: %v, Profit Factor: %.6f\n", opportunity.Triangle, opportunity.Profit)
	}
}

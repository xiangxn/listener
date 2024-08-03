package main

import (
	"fmt"
	"math"
	"testing"
)

// 构建邻接矩阵
func buildAdjacencyMatrix(pairs []Pair, nodes map[string]int) [][]float64 {
	n := len(nodes)
	adjMatrix := make([][]float64, n)
	for i := range adjMatrix {
		adjMatrix[i] = make([]float64, n)
		for j := range adjMatrix[i] {
			adjMatrix[i][j] = math.Inf(1)
		}
		adjMatrix[i][i] = 0
	}
	for _, pair := range pairs {
		from := nodes[pair.From]
		to := nodes[pair.To]
		adjustedPrice := pair.Price * (1 - pair.Fee)
		adjMatrix[from][to] = -math.Log(adjustedPrice)
		adjMatrix[to][from] = math.Log(1 / adjustedPrice)
	}
	return adjMatrix
}

// Floyd-Warshall 算法检测负权重环
func findArbitrage(adjMatrix [][]float64, nodes map[int]string) {
	n := len(adjMatrix)
	dist := make([][]float64, n)
	next := make([][]int, n)
	for i := range dist {
		dist[i] = make([]float64, n)
		copy(dist[i], adjMatrix[i])
		next[i] = make([]int, n)
		for j := range next[i] {
			if i != j && adjMatrix[i][j] < math.Inf(1) {
				next[i][j] = j
			} else {
				next[i][j] = -1
			}
		}
	}

	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if dist[i][j] > dist[i][k]+dist[k][j] {
					dist[i][j] = dist[i][k] + dist[k][j]
					next[i][j] = next[i][k]
				}
			}
		}
	}

	for i := 0; i < n; i++ {
		if dist[i][i] < 0 {
			cycle := findCycle(i, next)
			// if isValidCycle(cycle) {
			fmt.Println("Arbitrage opportunity detected:")
			for _, v := range cycle {
				fmt.Printf("%s -> ", nodes[v])
			}
			fmt.Printf("%s\n", nodes[i])
			return
			// }
		}
	}
	fmt.Println("No arbitrage opportunity found")
}

// 查找循环路径
func findCycle(start int, next [][]int) []int {
	path := []int{start}
	for x := next[start][start]; x != start && x != -1; x = next[x][start] {
		path = append(path, x)
		if len(path) > len(next) {
			break
		}
	}
	path = append(path, start) // 回到起点
	return path
}

// 验证路径是否有效
func isValidCycle(path []int) bool {
	if len(path) < 4 { // 至少三个不同的节点
		return false
	}
	visited := make(map[int]bool)
	for _, node := range path {
		if visited[node] {
			return false
		}
		visited[node] = true
	}
	return true
}

func TestTriangles2(t *testing.T) {
	// 示例数据
	pairs := []Pair{
		{"A", "B", 2.0, 0.001},
		{"B", "C", 1.5, 0.001},
		{"C", "A", 0.34, 0.001}, // 调整此处价格以确保存在套利
		{"A", "C", 1.7, 0.001},
		{"C", "B", 0.65, 0.001},
		{"B", "A", 0.5, 0.001},
	}

	// 构建节点映射
	nodes := make(map[string]int)
	nodesReverse := make(map[int]string)
	index := 0
	for _, pair := range pairs {
		if _, exists := nodes[pair.From]; !exists {
			nodes[pair.From] = index
			nodesReverse[index] = pair.From
			index++
		}
		if _, exists := nodes[pair.To]; !exists {
			nodes[pair.To] = index
			nodesReverse[index] = pair.To
			index++
		}
	}

	// 构建邻接矩阵
	adjMatrix := buildAdjacencyMatrix(pairs, nodes)

	// 查找潜在的套利机会
	findArbitrage(adjMatrix, nodesReverse)
}

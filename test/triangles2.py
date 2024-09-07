from collections import defaultdict
import math


def find_triangular_arbitrage_optimized(pairs):
    graph = defaultdict(dict)

    # 构建图
    for pair in pairs:
        tokenA = pair['tokenA']
        tokenB = pair['tokenB']
        priceAB = pair['priceAB']
        reserveA = pair['reserveA']
        reserveB = pair['reserveB']

        # 在图中添加正向和反向边
        graph[tokenA][tokenB] = { 'price': priceAB, 'reserve_in': reserveA, 'reserve_out': reserveB }
        graph[tokenB][tokenA] = { 'price': 1 / priceAB, 'reserve_in': reserveB, 'reserve_out': reserveA }

    arbitrage_opportunities = []

    # 遍历每个可能的三角路径
    for tokenA in graph:
        for tokenB in graph[tokenA]:
            for tokenC in graph[tokenB]:
                if tokenC in graph[tokenA]:
                    # 计算三角路径的回报率
                    amountB = calculate_amount_out(1, graph[tokenA][tokenB]['reserve_in'], graph[tokenA][tokenB]['reserve_out'])
                    amountC = calculate_amount_out(amountB, graph[tokenB][tokenC]['reserve_in'], graph[tokenB][tokenC]['reserve_out'])
                    finalAmountA = calculate_amount_out(amountC, graph[tokenC][tokenA]['reserve_in'], graph[tokenC][tokenA]['reserve_out'])

                    if finalAmountA > 1:
                        profit = finalAmountA - 1
                        arbitrage_opportunities.append({ "path": f"{tokenA} -> {tokenB} -> {tokenC} -> {tokenA}", "profit": profit })

    return arbitrage_opportunities


# 计算给定输入和储备量的输出
def calculate_amount_out(amount_in, reserve_in, reserve_out):
    amount_in_with_fee = amount_in * 997
    numerator = amount_in_with_fee * reserve_out
    denominator = reserve_in*1000 + amount_in_with_fee
    return numerator / denominator


# 示例交易对数据
pairs = [{
    "tokenA": "A",
    "tokenB": "B",
    "priceAB": 1.2,
    "reserveA": 1000,
    "reserveB": 1200
}, {
    "tokenA": "B",
    "tokenB": "C",
    "priceAB": 0.9,
    "reserveA": 1200,
    "reserveB": 1080
}, {
    "tokenA": "C",
    "tokenB": "A",
    "priceAB": 0.95,
    "reserveA": 1080,
    "reserveB": 1026
}, {
    "tokenA": "A",
    "tokenB": "C",
    "priceAB": 1.05,
    "reserveA": 1000,
    "reserveB": 1050
}, {
    "tokenA": "C",
    "tokenB": "B",
    "priceAB": 1.1,
    "reserveA": 1050,
    "reserveB": 1155
}, {
    "tokenA": "B",
    "tokenB": "A",
    "priceAB": 0.85,
    "reserveA": 1200,
    "reserveB": 1020
}]

opportunities = find_triangular_arbitrage_optimized(pairs)
for opp in opportunities:
    print(f"Arbitrage Opportunity: {opp['path']}, Profit: {opp['profit']}")

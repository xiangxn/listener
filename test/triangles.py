import pandas as pd
import numpy as np

# 假设您的数据表如下（这是一个示例数据表）
data = { 'pair': ['A/B', 'B/C', 'C/A', 'A/C', 'C/B', 'B/A'], 'price': [3.0, 1.5, 0.6, 1.7, 0.65, 0.3] }

# 创建数据框
df = pd.DataFrame(data)

# 构建价格图
graph = {}
for index, row in df.iterrows():
    pair = row['pair'].split('/')
    price = row['price']
    if pair[0] not in graph:
        graph[pair[0]] = {}
    if pair[1] not in graph:
        graph[pair[1]] = {}
    ap = price * (1-0.001)
    graph[pair[0]][pair[1]] = -np.log(ap)
    graph[pair[1]][pair[0]] = np.log(1 / ap)

# 寻找三角形路径
triangles = []
for start in graph:
    for mid in graph[start]:
        if mid == start:
            continue
        for end in graph[mid]:
            if end == start or end == mid:
                continue
            if start in graph[end]:
                triangles.append((start, mid, end))

# 计算潜在套利
arbitrage_opportunities = []
for triangle in triangles:
    weight = graph[triangle[0]][triangle[1]] + graph[triangle[1]][triangle[2]] + graph[triangle[2]][triangle[0]]
    if weight < 0:
        arbitrage_opportunities.append((triangle, np.exp(-weight)))

# 输出潜在套利路径
for opportunity in arbitrage_opportunities:
    print(f"Arbitrage opportunity: {opportunity[0]}, Profit Factor: {opportunity[1]}")

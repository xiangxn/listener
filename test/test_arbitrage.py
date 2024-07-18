# 定义初始池的状态
fee_v2 = 0.003
fee_v3 = 0.003
# 数据一
x2 = 40  # Uniswap V2中代币A的数量
y2 = 140000  # Uniswap V2中代币B的数量
x3 = 40  # Uniswap V3中代币A的数量
y3 = 142400  # Uniswap V3中代币B的数量

# 数据二
# x2 = 40
# y2 = 142400
# x3 = 40
# y3 = 140000

# 计算当前价格
price_v2 = y2 / x2
price_v3 = y3 / x3

# 计算目标价格
target_price = (price_v2+price_v3) / 2

# 计算V2中需要买入的数量
# 如果目标价格高于当前价格，则买入，否则卖出
if target_price > price_v2:
    delta_x2 = (target_price-price_v2) * x2 / target_price / 2
else:
    delta_x2 = (price_v2-target_price) * x2 / target_price / 2

delta_y2 = delta_x2 * target_price

print(f"price_v2: {price_v2}, price_v3: {price_v3}, target_price: {target_price}")

print(f"在Uniswap V2中买入的代币A数量: {delta_x2}")
print(f"在Uniswap V2中卖出的代币B数量: {delta_y2}")

# 计算V3中需要卖出的数量
# 如果目标价格低于当前价格，则卖出，否则买入
if target_price < price_v3:
    delta_x3 = (price_v3-target_price) * x3 / target_price / 2
else:
    delta_x3 = (target_price-price_v3) * x3 / target_price / 2

delta_y3 = delta_x3 * target_price

print(f"在Uniswap V3中卖出的代币A数量: {delta_x3}")
print(f"在Uniswap V3中买入的代币B数量: {delta_y3}")

#######################验证交易后价格#######################
if target_price < price_v2:
    new_x2 = x2 + delta_x2
    new_y2 = y2 - delta_y2
    new_x3 = x3 - delta_x3
    new_y3 = y3 + delta_y3
elif target_price > price_v2:
    new_x2 = x2 - delta_x2
    new_y2 = y2 + delta_y2
    new_x3 = x3 + delta_x3
    new_y3 = y3 - delta_y3

new_v2_price = new_y2 / new_x2
print("new v2 price: ", new_v2_price)
new_v3_price = new_y3 / new_x3
print("new v3 price: ", new_v3_price)

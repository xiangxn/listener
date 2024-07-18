from decimal import Decimal, getcontext
# 常数 Q96
Q96 = 2**96


def get_low_high_tick(tick, tick_spacing):
    """根据给定tick计算出下上边界tick"""
    tick_lower = tick // tick_spacing * tick_spacing
    tick_upper = tick_lower + tick_spacing
    return tick_lower, tick_upper


def tick_to_sqrt_price_x96(tick):
    abs_tick = abs(tick)
    ratio = 0xfffcb933bd6fad37aa2d162d1a594001 if (abs_tick & 0x1 != 0) else 0x100000000000000000000000000000000
    if (abs_tick & 0x2 != 0):
        ratio = (ratio * 0xfff97272373d413259a46990580e213a) >> 128
    if (abs_tick & 0x4 != 0):
        ratio = (ratio * 0xfff2e50f5f656932ef12357cf3c7fdcc) >> 128
    if (abs_tick & 0x8 != 0):
        ratio = (ratio * 0xffe5caca7e10e4e61c3624eaa0941cd0) >> 128
    if (abs_tick & 0x10 != 0):
        ratio = (ratio * 0xffcb9843d60f6159c9db58835c926644) >> 128
    if (abs_tick & 0x20 != 0):
        ratio = (ratio * 0xff973b41fa98c081472e6896dfb254c0) >> 128
    if (abs_tick & 0x40 != 0):
        ratio = (ratio * 0xff2ea16466c96a3843ec78b326b52861) >> 128
    if (abs_tick & 0x80 != 0):
        ratio = (ratio * 0xfe5dee046a99a2a811c461f1969c3053) >> 128
    if (abs_tick & 0x100 != 0):
        ratio = (ratio * 0xfcbe86c7900a88aedcffc83b479aa3a4) >> 128
    if (abs_tick & 0x200 != 0):
        ratio = (ratio * 0xf987a7253ac413176f2b074cf7815e54) >> 128
    if (abs_tick & 0x400 != 0):
        ratio = (ratio * 0xf3392b0822b70005940c7a398e4b70f3) >> 128
    if (abs_tick & 0x800 != 0):
        ratio = (ratio * 0xe7159475a2c29b7443b29c7fa6e889d9) >> 128
    if (abs_tick & 0x1000 != 0):
        ratio = (ratio * 0xd097f3bdfd2022b8845ad8f792aa5825) >> 128
    if (abs_tick & 0x2000 != 0):
        ratio = (ratio * 0xa9f746462d870fdf8a65dc1f90e061e5) >> 128
    if (abs_tick & 0x4000 != 0):
        ratio = (ratio * 0x70d869a156d2a1b890bb3df62baf32f7) >> 128
    if (abs_tick & 0x8000 != 0):
        ratio = (ratio * 0x31be135f97d08fd981231505542fcfa6) >> 128
    if (abs_tick & 0x10000 != 0):
        ratio = (ratio * 0x9aa508b5b7a84e1c677de54f3e99bc9) >> 128
    if (abs_tick & 0x20000 != 0):
        ratio = (ratio * 0x5d6af8dedb81196699c329225ee604) >> 128
    if (abs_tick & 0x40000 != 0):
        ratio = (ratio * 0x2216e584f5fa1ea926041bedfe98) >> 128
    if (abs_tick & 0x80000 != 0):
        ratio = (ratio * 0x48a170391f7dc42444e8fa2) >> 128
    if (tick > 0):
        ratio = 0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff // ratio
    return (ratio >> 32) + (0 if ratio % (1 << 32) == 0 else 1)


def calc_amount0_delta(liq, pa, pb, round_up):
    if pa > pb:
        pa, pb = pb, pa
    numerator1 = liq * Q96
    numerator2 = pb - pa
    result = numerator1 * numerator2 // pa // pb
    if round_up and ((numerator1*numerator2) % pa > 0):
        result += 1
    return result


def calc_amount1_delta(liq, pa, pb, round_up):
    if pa > pb:
        pa, pb = pb, pa
    result = liq * (pb-pa) // Q96
    if round_up and ((liq * (pb-pa)) % Q96 > 0):
        result += 1
    return result


def calc_amount0(liq, pa, pb):
    return -calc_amount0_delta(-liq, pa, pb, False) if liq < 0 else calc_amount0_delta(liq, pa, pb, True)


def calc_amount1(liq, pa, pb):
    return -calc_amount1_delta(-liq, pa, pb, False) if liq < 0 else calc_amount1_delta(liq, pa, pb, True)


getcontext().prec = 200

# 计算出来精度与solidity不一致
def tick_to_sqrtPriceX96(tick: int) -> int:
    # 计算1.0001的tick次方
    ratio = Decimal(1.0001)**tick
    # 计算平方根
    sqrt_ratio = ratio.sqrt()
    # 转换为Q64.96格式
    sqrtPriceX96 = int(sqrt_ratio * (2**96))
    return sqrtPriceX96


# 参数
liquidity = 66835806823279760363  # 假设的流动性值
sqrtPriceX96 = 182711349204900817339797210
tick = -121450  # 当前价格刻度
tick_spacing = 200
decimals0 = 18
decimals1 = 18

tick_lower, tick_upper = get_low_high_tick(tick, tick_spacing)

print("ticks:", tick_lower, tick, tick_upper)

sqrt_price_current = sqrtPriceX96
sqrt_price_lower = tick_to_sqrt_price_x96(tick_lower)
sqrt_price_upper = tick_to_sqrt_price_x96(tick_upper)
print("sqrt_prices:", sqrt_price_lower, sqrt_price_upper)

reserve0 = calc_amount0(liquidity, sqrt_price_current, sqrt_price_upper)
reserve1 = calc_amount1(liquidity, sqrt_price_lower, sqrt_price_current)

print(f"Reserve0:{reserve0} {reserve0/10**decimals0:.20f}")
print(f"Reserve1:{reserve1} {reserve1/10**decimals1:.20f}")

m = tick_to_sqrtPriceX96(tick_lower)
print("m: ", m)

from web3 import Web3
from multicall import Call, Multicall


# 初始化Web3
infura_url = 'https://eth-mainnet.g.alchemy.com/v2/nfYH4sMk9yov8TyR61N9PboMutjtrBe7'
web3 = Web3(Web3.HTTPProvider(infura_url))

# Uniswap V3 Pool合约地址和ABI
pool_address = '0x4e68Ccd3E89f51C3074ca5072bbAC773960dFa36'  # 示例地址
pool_abi = """[
    {
        "inputs": [
            {
                "internalType": "int16",
                "name": "",
                "type": "int16"
            }
        ],
        "name": "tickBitmap",
        "outputs": [
            {
                "internalType": "uint256",
                "name": "",
                "type": "uint256"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "tickSpacing",
        "outputs": [
            {
                "internalType": "int24",
                "name": "",
                "type": "int24"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "token0",
        "outputs": [
            {
                "internalType": "address",
                "name": "",
                "type": "address"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "token1",
        "outputs": [
            {
                "internalType": "address",
                "name": "",
                "type": "address"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "liquidity",
        "outputs": [
            {
                "internalType": "uint128",
                "name": "",
                "type": "uint128"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [],
        "name": "slot0",
        "outputs": [
            {
                "internalType": "uint160",
                "name": "sqrtPriceX96",
                "type": "uint160"
            },
            {
                "internalType": "int24",
                "name": "tick",
                "type": "int24"
            },
            {
                "internalType": "uint16",
                "name": "observationIndex",
                "type": "uint16"
            },
            {
                "internalType": "uint16",
                "name": "observationCardinality",
                "type": "uint16"
            },
            {
                "internalType": "uint16",
                "name": "observationCardinalityNext",
                "type": "uint16"
            },
            {
                "internalType": "uint8",
                "name": "feeProtocol",
                "type": "uint8"
            },
            {
                "internalType": "bool",
                "name": "unlocked",
                "type": "bool"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "int24",
                "name": "",
                "type": "int24"
            }
        ],
        "name": "ticks",
        "outputs": [
            {
                "internalType": "uint128",
                "name": "liquidityGross",
                "type": "uint128"
            },
            {
                "internalType": "int128",
                "name": "liquidityNet",
                "type": "int128"
            },
            {
                "internalType": "uint256",
                "name": "feeGrowthOutside0X128",
                "type": "uint256"
            },
            {
                "internalType": "uint256",
                "name": "feeGrowthOutside1X128",
                "type": "uint256"
            },
            {
                "internalType": "int56",
                "name": "tickCumulativeOutside",
                "type": "int56"
            },
            {
                "internalType": "uint160",
                "name": "secondsPerLiquidityOutsideX128",
                "type": "uint160"
            },
            {
                "internalType": "uint32",
                "name": "secondsOutside",
                "type": "uint32"
            },
            {
                "internalType": "bool",
                "name": "initialized",
                "type": "bool"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    }
]
"""  # 省略，使用Uniswap V3 Pool ABI
erc20_abi = """[
    {
        "constant": true,
        "inputs": [
            {
                "name": "who",
                "type": "address"
            }
        ],
        "name": "balanceOf",
        "outputs": [
            {
                "name": "",
                "type": "uint256"
            }
        ],
        "payable": false,
        "stateMutability": "view",
        "type": "function"
    },
    {
        "constant": true,
        "inputs": [],
        "name": "decimals",
        "outputs": [
            {
                "name": "",
                "type": "uint256"
            }
        ],
        "payable": false,
        "stateMutability": "view",
        "type": "function"
    }
]
"""

# 获取池合约
pool_contract = web3.eth.contract(address=pool_address, abi=pool_abi)

# 获取储备量信息
slot0 = pool_contract.functions.slot0().call()
sqrtPriceX96 = slot0[0]
current_tick = slot0[1]

liquidity = pool_contract.functions.liquidity().call()
tick_spacing = pool_contract.functions.tickSpacing().call()

# 常数 Q96
Q96 = 2**96


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


def get_low_high_tick(tick, tick_spacing):
    """根据给定tick计算出下上边界tick"""
    tick_lower = tick // tick_spacing * tick_spacing
    tick_upper = tick_lower + tick_spacing
    return tick_lower, tick_upper


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


def calculate_reserves(tick_data, sqrtPriceX96, decimals0, decimals1):
    # print("tick_data:", tick_data)
    # 转换 sqrtPriceX96 为 Decimal 类型
    sqrt_price_current = sqrtPriceX96

    # 总储备量初始化
    reserve0_total = 0
    reserve1_total = 0
    # liquidity_current = liquidity
    liquidity_current = 0

    for tick, liquidity_net in tick_data.items():
        liquidity_current += liquidity_net
        if liquidity_net != 0:
            tick_lower, tick_upper = get_low_high_tick(tick, tick_spacing)

            sqrt_price_lower = tick_to_sqrt_price_x96(tick_lower)
            sqrt_price_upper = tick_to_sqrt_price_x96(tick_upper)

            if current_tick < tick_lower:
                reserve0 = calc_amount0(liquidity_current, sqrt_price_lower, sqrt_price_upper)
                reserve0_total += reserve0
            elif current_tick < tick_upper:
                reserve0 = calc_amount0(liquidity_current, sqrt_price_current, sqrt_price_upper)
                reserve1 = calc_amount1(liquidity_current, sqrt_price_lower, sqrt_price_current)
                reserve0_total += reserve0
                reserve1_total += reserve1
            else:
                reserve1 = calc_amount1(liquidity_current, sqrt_price_lower, sqrt_price_upper)
                reserve1_total += reserve1

    return reserve0_total, reserve1_total


def get_all_active_ticks(pool_contract):

    # 确定索引范围，-887272 到 887272 之间
    tick_bitmap_indices = range(-887272 >> 8, (887272 >> 8) + 1)
    print("tick_bitmap_indices count:", len(tick_bitmap_indices))

    # 使用 batch 请求获取所有 tickBitmap 数据
    active_ticks = []
    batch_size = 5000  # 每批处理的请求数量
    for i in range(0, len(tick_bitmap_indices), batch_size):
        batch_indices = tick_bitmap_indices[i:i + batch_size]
        multicall_bitmap = Multicall([Call(pool_contract.address, ['tickBitmap(int16)(int256)', index], [[index, None]]) for index in batch_indices], _w3=web3)

        tick_bitmap_data = multicall_bitmap()

        for index, bitmap in tick_bitmap_data.items():
            if bitmap:
                for j in range(256):
                    if bitmap & (1 << j):
                        tick = (index*256 + j) * tick_spacing
                        active_ticks.append(tick)

    return active_ticks


def get_tick_data(pool_contract, ticks):
    # 使用 batch 请求获取所有 ticks 数据
    tick_data = {}
    batch_size = 5000  # 每批处理的请求数量
    for i in range(0, len(ticks), batch_size):
        batch_ticks = ticks[i:i + batch_size]
        multicall = Multicall([
            Call(pool_contract.address, ['ticks(int24)((uint128,int128,uint256,uint256,int56,uint160,uint32,bool))', tick], [[tick, None]])
            for tick in batch_ticks
        ],
                              _w3=web3)

        responses = multicall()

        # 创建一个字典以 tick 为键，liquidityNet 为值
        for tick, response in responses.items():
            if response:
                tick_data[tick] = response[1]

    return tick_data


def get_balances(pool_contract):
    token0 = pool_contract.functions.token0().call()
    token1 = pool_contract.functions.token1().call()

    token0_contract = web3.eth.contract(address=token0, abi=erc20_abi)
    token1_contract = web3.eth.contract(address=token1, abi=erc20_abi)
    decimals0 = token0_contract.functions.decimals().call()
    decimals1 = token1_contract.functions.decimals().call()

    balance0 = token0_contract.functions.balanceOf(pool_address).call()
    balance1 = token1_contract.functions.balanceOf(pool_address).call()
    return balance0, balance1, decimals0, decimals1


def main():
    (balance0, balance1, decimals0, decimals1) = get_balances(pool_contract)
    print("balances: ", balance0 / 10**decimals0, balance1 / 10**decimals1, decimals0, decimals1)

    # 获取所有活跃的 ticks
    ticks = get_all_active_ticks(pool_contract)
    print("active_ticks_count:", len(ticks))
    # print("active_ticks:", ticks)

    # 获取所有活跃 ticks 的流动性数据
    tick_data = get_tick_data(pool_contract, ticks)
    print("tick_data_count:", len(tick_data))
    # print("tick_data:", tick_data)

    # 计算储备量
    reserve0, reserve1 = calculate_reserves(tick_data, sqrtPriceX96, decimals0, decimals1)

    print(f"Reserve0: {reserve0/10**decimals0}")
    print(f"Reserve1: {reserve1/10**decimals1}")


# 运行主函数
main()

import os
from dotenv import load_dotenv
from web3 import Web3
from web3._utils.events import get_event_data, event_abi_to_log_topic

load_dotenv()

# 连接到以太坊节点
web3 = Web3(Web3.HTTPProvider(os.getenv("RPC_MAINNET")))

# 合约地址和ABI
contract_address = '0x75001b3FfE0f77864c7Dc64c55e1E22b205e4a07'
contract_abi = [{
    "anonymous":
    False,
    "inputs": [{
        "indexed": True,
        "internalType": "address",
        "name": "sender",
        "type": "address"
    }, {
        "indexed": False,
        "internalType": "uint256",
        "name": "amount0In",
        "type": "uint256"
    }, {
        "indexed": False,
        "internalType": "uint256",
        "name": "amount1In",
        "type": "uint256"
    }, {
        "indexed": False,
        "internalType": "uint256",
        "name": "amount0Out",
        "type": "uint256"
    }, {
        "indexed": False,
        "internalType": "uint256",
        "name": "amount1Out",
        "type": "uint256"
    }, {
        "indexed": True,
        "internalType": "address",
        "name": "to",
        "type": "address"
    }],
    "name":
    "Swap",
    "type":
    "event"
}]

# 创建合约实例
contract = web3.eth.contract(address=contract_address, abi=contract_abi)

event_signature = event_abi_to_log_topic(contract.events.Swap().abi)  # 替换为实际的事件名称


# 获取事件Logs
def get_event_logs(from_block, to_block):
    # 创建过滤器
    event_filter = web3.eth.filter({ 'fromBlock': from_block, 'toBlock': to_block, 'address': contract_address, 'topics': [event_signature] })
    logs = event_filter.get_all_entries()
    return logs


# 示例调用
from_block = 20361539
to_block = 'latest'
logs = get_event_logs(from_block, to_block)

# 打印日志
for log in logs:
    print(log, "\n")

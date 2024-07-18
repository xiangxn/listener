from web3 import Web3, HTTPProvider


def impersonate_account(web3: Web3, address: str):
    """
    Impersonate account through Anvil without needing private key
    :param address:
        Account to impersonate
    """
    web3.provider.make_request("anvil_impersonateAccount", [address])


if __name__ == "__main__":
    PROVIDER = "http://127.0.0.1:8540"
    provider = HTTPProvider(PROVIDER)
    web3 = Web3(provider)
    impersonate_account(web3, "0xF04a5cC80B1E94C69B48f5ee68a08CD2F09A7c3E")

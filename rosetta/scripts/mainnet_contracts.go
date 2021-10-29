package scripts

var mainnetContracts = map[string]string{
	"c6c77b9f5c7a378f": `// This script reads the Flow balance of FlowSwapPair contract as well as standard Flow vault balance

		import FungibleToken from 0x{{.Params.FungibleToken}}
		import {{.Token.Type}} from 0x{{.Token.Address}}

		import FlowSwapPair from 0xc6c77b9f5c7a378f //parametrize address - does it exist on other chains?

		pub fun main(account: Address): UFix64 {

			let vaultRef = getAccount(account)
				.getCapability({{.Token.Balance}})
				.borrow<&{{.Token.Type}}.Vault{FungibleToken.Balance}>()
				?? panic("Could not borrow Balance reference to the Vault")
		
			return vaultRef.balance + FlowSwapPair.getPoolAmounts().token1Amount
		}
	`,
}

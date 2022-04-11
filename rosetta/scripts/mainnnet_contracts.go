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
	"d796ff17107bbff6": `
		import FungibleToken from 0x{{.Params.FungibleToken}}
		import {{.Token.Type}} from 0x{{.Token.Address}}
		import Versus from 0xd796ff17107bbff6
		
		pub fun main(account: Address): UFix64 {
		
			// default flow token
			let vaultRef = getAccount(account)
				.getCapability({{.Token.Balance}})
				.borrow<&{{.Token.Type}}.Vault{FungibleToken.Balance}>()
				?? panic("Could not borrow Balance reference to the Vault")
		
		
			// sum of all public drops
			let publicDrop = getAccount(0xd796ff17107bbff6)
				.getCapability(Versus.CollectionPublicPath)
				.borrow<&{Versus.PublicDrop}>()
				?? panic("Could not borrow Balance reference to the PublicDrop")
		
			let allStatuses = publicDrop.getAllStatuses()
		
			var totalFlow = 0.0
		
			for dropId in allStatuses.keys {
				let dropStatus = allStatuses[dropId]
				totalFlow = totalFlow + (dropStatus?.uniquePrice ?? 0.0) + (dropStatus?.editionPrice ?? 0.0)
			}
		
			return totalFlow + vaultRef.balance
		}
	`,
}
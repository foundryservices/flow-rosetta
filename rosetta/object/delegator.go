// Foundry Rosetta Fork: Delegator
package object

// Delegator Delegator is the identifier and value of a wallet delegating to a validator.
type Delegator struct {
	// Wallet address for the delegator.
	Address string `json:"address"`
	// Value of all the wallet transactions.
	Value string `json:"value"`
	// Value of the delegated wallet transactions.
	DelegatedValue string `json:"delegated_value"`
}
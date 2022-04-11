package scripts

type StakingNodeInfo struct {
	Delegators []struct {
		ID                       string `json:"id"`
		NodeID                   string `json:"nodeID"`
		TokensCommitted          string `json:"tokensCommitted"`
		TokensRequestedToUnstake string `json:"tokensRequestedToUnstake"`
		TokensRewarded           string `json:"tokensRewarded"`
		TokensStaked             string `json:"tokensStaked"`
		TokensUnstaked           string `json:"tokensUnstaked"`
		TokensUnstaking          string `json:"tokensUnstaking"`
	} `json:"delegators"`
	Node struct {
		DelegatorIDCounter       string   `json:"delegatorIDCounter"`
		Delegators               []string `json:"delegators"`
		ID                       string   `json:"id"`
		InitialWeight            string   `json:"initialWeight"`
		NetworkingAddress        string   `json:"networkingAddress"`
		NetworkingKey            string   `json:"networkingKey"`
		Role                     string   `json:"role"`
		StakingKey               string   `json:"stakingKey"`
		TokensCommitted          string   `json:"tokensCommitted"`
		TokensRequestedToUnstake string   `json:"tokensRequestedToUnstake"`
		TokensRewarded           string   `json:"tokensRewarded"`
		TokensStaked             string   `json:"tokensStaked"`
		TokensUnstaked           string   `json:"tokensUnstaked"`
		TokensUnstaking          string   `json:"tokensUnstaking"`
	} `json:"node"`
	StakedBalance string `json:"stakedBalance"`
}

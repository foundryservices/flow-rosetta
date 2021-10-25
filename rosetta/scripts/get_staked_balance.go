// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package scripts

// Adopted from:
// https://github.com/onflow/flow-core-contracts/blob/master/transactions/flowToken/scripts/get_balance.cdc
// https://github.com/onflow/flow-core-contracts/blob/master/contracts/FlowIDTableStaking.cdc

const getStakedBalance = `
// This script reads the balance field of an account's FlowToken Balance
// and total balance of all staked nodes with their delegators

import FungibleToken from 0x{{.Params.FungibleToken}}
import {{.Token.Type}} from 0x{{.Token.Address}}
import FlowIDTableStaking from 0x{{.Params.StakingTable}}


pub fun main(account: Address): UFix64 {

    let vaultRef = getAccount(account)
        .getCapability({{.Token.Balance}})
        .borrow<&{{.Token.Type}}.Vault{FungibleToken.Balance}>()
        ?? panic("Could not borrow Balance reference to the Vault")

	let vaultBalance = vaultRef.balance

	// Sum up all tokens from all delegators and all stake
	let allNodeIDs = FlowIDTableStaking.getNodeIDs()

    var totalTokens: UFix64 = 0.0

    for nodeID in allNodeIDs {
        let nodeInfo = FlowIDTableStaking.NodeInfo(nodeID: nodeID)
        let delegatorsIDs = nodeInfo.delegators

        totalTokens = totalTokens + nodeInfo.totalTokensInRecord()

        for delegatorID in delegatorsIDs {
            let delegatorInfo = FlowIDTableStaking.DelegatorInfo(nodeID: nodeID, delegatorID: delegatorID)


            totalTokens = totalTokens + delegatorInfo.totalTokensInRecord()
        }
    }

    return vaultBalance + stakedBalance
}
`

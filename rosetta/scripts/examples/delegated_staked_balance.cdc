// This script reads the balance field of an account's FlowToken Balance
// and total balance of all staked nodes with their delegators
import FlowIDTableStaking from 0x8624b52f9ddcd04a

pub struct StakingNodeInfo {
    pub var stakedBalance: UFix64
    pub var node: FlowIDTableStaking.NodeInfo
    pub var delegators: [FlowIDTableStaking.DelegatorInfo]

    init (nodeID: String) {
        self.node = FlowIDTableStaking.NodeInfo(nodeID: nodeID)
        self.stakedBalance = self.node.totalTokensInRecord()
        self.delegators = []

        for delegatorID in self.node.delegators {
            let delegatorInfo = FlowIDTableStaking.DelegatorInfo(nodeID: self.node.id, delegatorID: delegatorID)
            self.delegators.append(delegatorInfo)
            self.stakedBalance = self.stakedBalance + delegatorInfo.totalTokensInRecord()
        }
    }
}

pub fun main(nodeID: String): StakingNodeInfo {
    return StakingNodeInfo(nodeID: nodeID)
}
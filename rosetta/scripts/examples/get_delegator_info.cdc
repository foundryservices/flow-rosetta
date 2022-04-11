import FlowIDTableStaking from 0x8624b52f9ddcd04a
// flow scripts execute -n mainnet -l=debug -o=json -s=./out/result.json get_delegator_info.cdc 
pub fun main(): FlowIDTableStaking.DelegatorInfo {
  var nodeID = getStakedNodeIDs()[0]
  nodeID = "e74d0ca7afed2b5bfd612f3c75010adb229d632ca4ab3b77d178ee226d7ec2a2"
  var delegatorID = getNodeInfo(nodeID: nodeID).delegators[0]
  return getDelegatorInfo(nodeID: nodeID, delegatorID: delegatorID)
}

pub fun getStakedNodeIDs(): [String] {
  let nodeIDs = FlowIDTableStaking.getStakedNodeIDs()
  return nodeIDs
}

pub fun getNodeIDs(): [String] {
  let nodeIDs = FlowIDTableStaking.getNodeIDs()
  return nodeIDs
}

pub fun getNodeInfo(nodeID: String): FlowIDTableStaking.NodeInfo {
  let nodeInfo = FlowIDTableStaking.NodeInfo(nodeID)
  return nodeInfo
}

pub fun getDelegatorInfo(nodeID: String, delegatorID: UInt32): FlowIDTableStaking.DelegatorInfo {
  let delegatorInfo = FlowIDTableStaking.DelegatorInfo(nodeID, delegatorID)
  return delegatorInfo
}
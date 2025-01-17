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

package transactor

import (
	sdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-rosetta/rosetta/identifier"
)

func rosettaTxID(txID sdk.Identifier) identifier.Transaction {
	return identifier.Transaction{
		Hash: txID.String(),
	}
}

func rosettaBlockID(height uint64, blockID flow.Identifier) identifier.Block {
	return identifier.Block{
		Index: &height,
		Hash:  blockID.String(),
	}
}

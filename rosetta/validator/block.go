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

package validator

import (
	"context"
	"fmt"

	sdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-rosetta/rosetta/failure"
	"github.com/optakt/flow-rosetta/rosetta/identifier"
)

// Block tries to extrapolate the block identifier to a full version
// of itself. If both index and hash are zero values, it is assumed that the
// latest block is referenced.
func (v *Validator) Block(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {

	// If both the index and the hash are missing, the block identifier is invalid, and
	// the latest block ID is returned instead.
	if rosBlockID.Index == nil && rosBlockID.Hash == "" {
		last, err := v.accessAPI.GetLatestBlockHeader(context.Background(), true) // FIXME should we use isSealed?
		// last, err := v.index.Last()
		if err != nil {
			return 0, flow.ZeroID, fmt.Errorf("could not retrieve last: %w", err)
		}
		// header, err := v.index.Header(last)
		// if err != nil {
		// 	return 0, flow.ZeroID, fmt.Errorf("could not retrieve header: %w", err)
		// }
		return last.Height, flow.Identifier(last.ID), nil
	}

	// If a block hash is present, it should be a valid block ID for Flow.
	if rosBlockID.Hash != "" {
		_, err := flow.HexStringToIdentifier(rosBlockID.Hash)
		if err != nil {
			return 0, flow.ZeroID, failure.InvalidBlock{
				Description: failure.NewDescription(blockInvalid,
					failure.WithString("block_hash", rosBlockID.Hash),
				),
			}
		}
	}

	// If a block index is present, it should be a valid height for the DPS.
	if rosBlockID.Index != nil {
		// first, err := v.index.First()
		// if err != nil {
		// 	return 0, flow.ZeroID, fmt.Errorf("could not get first: %w", err)
		// }
		var first uint64 = 0 // FIXME: might be able to get from accessAPI
		if *rosBlockID.Index < first {
			return 0, flow.ZeroID, failure.InvalidBlock{
				Description: failure.NewDescription(blockTooLow,
					failure.WithUint64("block_index", *rosBlockID.Index),
					failure.WithUint64("first_index", first),
				),
			}
		}
		// last, err := v.index.Last()
		last, err := v.accessAPI.GetLatestBlockHeader(context.Background(), true)
		if err != nil {
			return 0, flow.ZeroID, fmt.Errorf("could not get last: %w", err)
		}
		if *rosBlockID.Index > last.Height {
			return 0, flow.ZeroID, failure.UnknownBlock{
				Index: *rosBlockID.Index,
				Hash:  rosBlockID.Hash,
				Description: failure.NewDescription(blockTooHigh,
					failure.WithUint64("last_index", last.Height),
				),
			}
		}
	}

	// If we don't have a height, fill it in now.
	if rosBlockID.Index == nil {
		blockID := sdk.HexToID(rosBlockID.Hash)
		// blockID, _ := flow.HexStringToIdentifier(rosBlockID.Hash)
		height, err := v.accessAPI.GetBlockHeaderByID(context.Background(), blockID)
		//height, err := v.index.HeightForBlock(blockID)
		if err != nil {
			return 0, flow.ZeroID, fmt.Errorf("could not get height for block: %w", err)
		}
		rosBlockID.Index = &height.Height
	}

	// The given block ID should match the block ID at the given height.
	header, err := v.accessAPI.GetBlockHeaderByHeight(context.Background(), *rosBlockID.Index)
	// header, err := v.index.Header(*rosBlockID.Index)
	if err != nil {
		return 0, flow.ZeroID, fmt.Errorf("could not get header: %w", err)
	}
	if rosBlockID.Hash != "" && rosBlockID.Hash != header.ID.Hex() {
		return 0, flow.ZeroID, failure.InvalidBlock{
			Description: failure.NewDescription(blockMismatch,
				failure.WithUint64("block_index", *rosBlockID.Index),
				failure.WithString("block_hash", rosBlockID.Hash),
				failure.WithString("want_hash", header.ID.Hex()),
			),
		}
	}

	identifier, err := flow.HexStringToIdentifier(header.ID.Hex())
	if err != nil {
		return 0, flow.ZeroID, fmt.Errorf("could not convert to identifier: %w", err)
	}

	return header.Height, identifier, nil
}

// CompleteBlockID verifies that both index and hash are populated in the block ID.
func (v *Validator) CompleteBlockID(rosBlockID identifier.Block) error {
	if rosBlockID.Index == nil || rosBlockID.Hash == "" {
		return failure.IncompleteBlock{
			Description: failure.NewDescription(blockNotFull),
		}
	}
	return nil
}

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

package object

import (
	"github.com/optakt/flow-rosetta/rosetta/identifier"
)

// Amount is some value of a currency. An amount must have both a value and a currency.
type Amount struct {
	Value    string              `json:"value"`
	Currency identifier.Currency `json:"currency"`

	// Foundry Rosetta Fork: The total delegated portion of the Value on a validator. Null if account is not a validator.
	DelegatedValue string `json:"delegated_value,omitempty"`
	// Foundry Rosetta Fork: A list of delegators for a given validator and the value delegated. Null if account is not a
	// validator.
	Delegators []*Delegator `json:"delegators,omitempty"`
}

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

import (
	"bytes"
	"fmt"
	"github.com/onflow/flow-go/model/flow"
	"text/template"

	"github.com/optakt/flow-dps/models/dps"
)

// Generator dynamically generates Cadence scripts from templates.
type Generator struct {
	params          dps.Params
	getBalance      *template.Template
	getStakedBalance *template.Template
	transferTokens  *template.Template
	tokensDeposited *template.Template
	tokensWithdrawn *template.Template
	delegatorRewardsPaid *template.Template
	custom           map[flow.ChainID]map[flow.Address]*template.Template
}

// NewGenerator returns a Generator using the given parameters.
func NewGenerator(params dps.Params) *Generator {
	g := Generator{
		params:          params,
		getBalance:      template.Must(template.New("get_balance").Parse(getBalance)),
		getStakedBalance: template.Must(template.New("get_staked_balance").Parse(getStakedBalance)),
		transferTokens:  template.Must(template.New("transfer_tokens").Parse(transferTokens)),
		tokensDeposited: template.Must(template.New("tokensDeposited").Parse(tokensDeposited)),
		tokensWithdrawn: template.Must(template.New("withdrawal").Parse(tokensWithdrawn)),
		delegatorRewardsPaid: template.Must(template.New("delegator_rewards_paid").Parse(delegatorRewardsPaid)),
		custom:           map[flow.ChainID]map[flow.Address]*template.Template{},
	}

	var mainnetCustom = make(map[flow.Address]*template.Template)

	for address, contract := range mainnetContracts {
		mainnetCustom[flow.HexToAddress(address)] = template.Must(template.New(fmt.Sprintf("mainnet_%s", address)).Parse(contract))
	}
	g.custom[flow.Mainnet] = mainnetCustom

	return &g
}

// GetBalance generates a Cadence script to retrieve the balance of an account.
func (g *Generator) GetBalance(symbol string) ([]byte, error) {
	return g.bytes(g.getBalance, symbol)
}

// GetStakedBalance generates a Cadence script to retrieve the balance of an account with
func (g *Generator) GetStakedBalance(symbol string) ([]byte, error) {
	return g.bytes(g.getStakedBalance, symbol)
}

// TransferTokens generates a Cadence script to operate a token transfer transaction.
func (g *Generator) TransferTokens(symbol string) ([]byte, error) {
	return g.bytes(g.transferTokens, symbol)
}

// TokensDeposited generates a Cadence script that matches the Flow event for tokens being deposited.
func (g *Generator) TokensDeposited(symbol string) (string, error) {
	return g.string(g.tokensDeposited, symbol)
}

// TokensWithdrawn generates a Cadence script that matches the Flow event for tokens being withdrawn.
func (g *Generator) TokensWithdrawn(symbol string) (string, error) {
	return g.string(g.tokensWithdrawn, symbol)
}

// DelegatorRewardsPaid generates a Cadence script that matches the Flow event for delegator rewards being paid.
func (g *Generator) DelegatorRewardsPaid(symbol string) (string, error) {
	return g.string(g.delegatorRewardsPaid, symbol)
}

func (g *Generator) Custom(symbol string, chainID flow.ChainID, address flow.Address) (bool, []byte, error) {

	var has bool

	chainCustom, has := g.custom[chainID]
	if !has {
		return false, nil, nil
	}
	template, has := chainCustom[address]
	if !has {
		return false, nil, nil
	}

	bytes, err := g.bytes(template, symbol)
	return true, bytes, err
}

func (g *Generator) string(template *template.Template, symbol string) (string, error) {
	buf, err := g.compile(template, symbol)
	if err != nil {
		return "", fmt.Errorf("could not compile template: %w", err)
	}
	return buf.String(), nil
}

func (g *Generator) bytes(template *template.Template, symbol string) ([]byte, error) {
	buf, err := g.compile(template, symbol)
	if err != nil {
		return nil, fmt.Errorf("could not compile template: %w", err)
	}
	return buf.Bytes(), nil
}

func (g *Generator) compile(template *template.Template, symbol string) (*bytes.Buffer, error) {
	token, ok := g.params.Tokens[symbol]
	if !ok {
		return nil, fmt.Errorf("invalid token symbol (%s)", symbol)
	}
	data := struct {
		Params dps.Params
		Token  dps.Token
	}{
		Params: g.params,
		Token:  token,
	}
	buf := &bytes.Buffer{}
	err := template.Execute(buf, data)
	if err != nil {
		return nil, fmt.Errorf("could not execute template: %w", err)
	}
	return buf, nil
}

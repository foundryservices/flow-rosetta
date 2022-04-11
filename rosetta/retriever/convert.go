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

package retriever

import (
	"encoding/json"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-rosetta/rosetta/identifier"
)

func rosettaTxID(txID flow.Identifier) identifier.Transaction {
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

func rosettaCurrency(symbol string, decimals uint) identifier.Currency {
	return identifier.Currency{
		Symbol:   symbol,
		Decimals: decimals,
	}
}

func valueToJsonString(value cadence.Value) string {
	result := flatten(value)
	json, _ := json.MarshalIndent(result, "", "    ")
	return string(json)
}

func flatten(field cadence.Value) interface{} {
	dictionaryValue, isDictionary := field.(cadence.Dictionary)
	structValue, isStruct := field.(cadence.Struct)
	arrayValue, isArray := field.(cadence.Array)
	if isStruct {
		subStructNames := structValue.StructType.Fields
		result := map[string]interface{}{}
		for j, subField := range structValue.Fields {
			result[subStructNames[j].Identifier] = flatten(subField)
		}
		return result
	} else if isDictionary {
		result := map[string]interface{}{}
		for _, item := range dictionaryValue.Pairs {
			result[item.Key.String()] = flatten(item.Value)
		}
		return result
	} else if isArray {
		result := []interface{}{}
		for _, item := range arrayValue.Values {
			result = append(result, flatten(item))
		}
		return result
	}
	result, err := strconv.Unquote(field.String())
	if err != nil {
		return field.String()
	}
	return result

}

// "1.00000000" -> "100000000"
func UFix64ToUInt64String(UFix64 string) (string, error) {
	ufix, err := cadence.NewUFix64(UFix64)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(ufix.ToGoValue().(uint64), 10), nil
}

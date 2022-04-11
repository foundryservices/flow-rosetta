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

//go:build integration
// +build integration

package rosetta_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/invoker"
	"github.com/optakt/flow-dps/service/storage"
	"github.com/optakt/flow-rosetta/api/rosetta"
	"github.com/optakt/flow-rosetta/rosetta/configuration"
	"github.com/optakt/flow-rosetta/rosetta/converter"
	"github.com/optakt/flow-rosetta/rosetta/identifier"
	"github.com/optakt/flow-rosetta/rosetta/meta"
	"github.com/optakt/flow-rosetta/rosetta/retriever"
	"github.com/optakt/flow-rosetta/rosetta/scripts"
	"github.com/optakt/flow-rosetta/rosetta/validator"
	"github.com/optakt/flow-rosetta/testing/snapshots"
)

const (
	balanceEndpoint     = "/account/balance"
	blockEndpoint       = "/block"
	transactionEndpoint = "/block/transaction"
	listEndpoint        = "/network/list"
	optionsEndpoint     = "/network/options"
	statusEndpoint      = "/network/status"

	invalidBlockchain = "invalid-blockchain"
	invalidNetwork    = "invalid-network"
	invalidToken      = "invalid-token"

	invalidBlockHash = "f91704ce2fa9a1513500184ebfec884a1728438463c0104f8a17d5c66dd1af7z" // invalid hex value
)

func setupDB(t *testing.T) *badger.DB {
	t.Helper()

	opts := badger.DefaultOptions("").
		WithInMemory(true).
		WithLogger(nil)

	db, err := badger.Open(opts)
	require.NoError(t, err)

	reader := hex.NewDecoder(strings.NewReader(snapshots.Rosetta))

	decompressor, err := zstd.NewReader(reader)
	require.NoError(t, err)

	err = db.Load(decompressor, runtime.GOMAXPROCS(0))
	require.NoError(t, err)

	return db
}

func setupAPI(t *testing.T, db *badger.DB) *rosetta.Data {
	t.Helper()

	rosetta.EnableSmartCodes()

	codec := zbor.NewCodec()
	storage := storage.New(codec)
	index := index.NewReader(db, storage)

	params := dps.FlowParams[dps.FlowLocalnet]
	config := configuration.New(params.ChainID)
	validate := validator.New(params, index, nil, config)
	generate := scripts.NewGenerator(params)
	invoke, err := invoker.New(index)
	require.NoError(t, err)
	convert, err := converter.New(generate)
	require.NoError(t, err)
	retrieve := retriever.New(params, index, validate, generate, invoke, convert)
	controller := rosetta.NewData(config, retrieve, validate)

	return controller
}

func setupRecorder(endpoint string, input interface{}, options ...func(*http.Request)) (*httptest.ResponseRecorder, echo.Context, error) {

	payload, ok := input.([]byte)
	if !ok {
		var err error
		payload, err = json.Marshal(input)
		if err != nil {
			return nil, echo.New().AcquireContext(), fmt.Errorf("could not encode input: %w", err)
		}
	}

	req := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	for _, opt := range options {
		opt(req)
	}

	rec := httptest.NewRecorder()

	ctx := echo.New().NewContext(req, rec)

	return rec, ctx, nil
}

func checkRosettaError(statusCode int, def meta.ErrorDefinition) func(t assert.TestingT, err error, v ...interface{}) bool {

	return func(t assert.TestingT, err error, v ...interface{}) bool {
		// return false if any of the asserts failed
		success := true
		success = success && assert.Error(t, err)

		if !assert.IsType(t, &echo.HTTPError{}, err) {
			return false
		}
		echoErr := err.(*echo.HTTPError)

		success = success && assert.Equal(t, statusCode, echoErr.Code)

		if !assert.IsType(t, rosetta.Error{}, echoErr.Message) {
			return false
		}

		gotErr := echoErr.Message.(rosetta.Error)

		success = success && assert.Equal(t, def, gotErr.ErrorDefinition)
		return success
	}
}

// defaultNetwork returns the Network identifier common for all requests.
func defaultNetwork() identifier.Network {
	return identifier.Network{
		Blockchain: dps.FlowBlockchain,
		Network:    dps.FlowLocalnet.String(),
	}
}

// defaultCurrency returns the Currency spec common for all requests.
// For now this only gets the FLOW tokens.
func defaultCurrency() []identifier.Currency {
	return []identifier.Currency{
		{
			Symbol:   dps.FlowSymbol,
			Decimals: dps.FlowDecimals,
		},
	}
}

func validateBlock(t *testing.T, height uint64, hash string) validateBlockFunc {
	t.Helper()

	return func(rosBlockID identifier.Block) {
		assert.Equal(t, hash, rosBlockID.Hash)
		require.NotNil(t, rosBlockID.Index)
		assert.Equal(t, height, *rosBlockID.Index)
	}
}

func validateByHeader(t *testing.T, header flow.Header) validateBlockFunc {
	return validateBlock(t, header.Height, header.ID().String())
}

func getUint64P(n uint64) *uint64 {
	return &n
}

func knownHeader(height uint64) flow.Header {

	switch height {

	case 0:
		return flow.Header{
			ChainID:        dps.FlowLocalnet,
			ParentID:       flow.Identifier{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			Height:         0,
			PayloadHash:    flow.Identifier{0x7b, 0x3b, 0x31, 0x3b, 0xd8, 0x3e, 0x1, 0xd1, 0x3c, 0x44, 0x9d, 0x4d, 0xd4, 0xba, 0xc0, 0x41, 0x37, 0xf5, 0x9, 0xb, 0xcb, 0x30, 0x5d, 0xdd, 0x75, 0x2, 0x98, 0xbd, 0x16, 0xe5, 0x33, 0x9b},
			Timestamp:      time.Unix(0, 1632143221831215000).UTC(),
			View:           0,
			ParentVoterIDs: []flow.Identifier{},

			ProposerID: flow.Identifier{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		}

	case 1:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0xf7, 0x67, 0xf0, 0xbd, 0xd, 0x22, 0x49, 0xf8, 0x6b, 0x35, 0x20, 0x7b, 0xa1, 0xb4, 0xdc, 0xf6, 0xb0, 0x65, 0x4f, 0xb6, 0xe, 0xf1, 0x8e, 0x5d, 0xec, 0x32, 0xa4, 0x35, 0x8f, 0x99, 0x81, 0xbb},
			Height:      1,
			PayloadHash: flow.Identifier{0x7b, 0x3b, 0x31, 0x3b, 0xd8, 0x3e, 0x1, 0xd1, 0x3c, 0x44, 0x9d, 0x4d, 0xd4, 0xba, 0xc0, 0x41, 0x37, 0xf5, 0x9, 0xb, 0xcb, 0x30, 0x5d, 0xdd, 0x75, 0x2, 0x98, 0xbd, 0x16, 0xe5, 0x33, 0x9b},
			Timestamp:   time.Unix(0, 1632143288474393800).UTC(),
			View:        2,
			ParentVoterIDs: []flow.Identifier{
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			},
			ParentVoterSigData: []byte{0xa8, 0xe7, 0xe6, 0xf, 0xd6, 0x65, 0xb5, 0x5b, 0xb7, 0x4e, 0x9d, 0xfc, 0xb8, 0xf1, 0xcb, 0xaf, 0xae, 0x52, 0xf9, 0x52, 0xeb, 0xf7, 0x89, 0x4c, 0x2e, 0x74, 0x73, 0x39, 0x2b, 0x25, 0x33, 0xb1, 0x2f, 0xf3, 0x1a, 0xa5, 0x55, 0xd5, 0xe6, 0x7b, 0x9f, 0xdd, 0x28, 0x62, 0x54, 0x8, 0x70, 0x42, 0xa1, 0x6f, 0x21, 0xd2, 0xf1, 0xa5, 0xc2, 0x30, 0x63, 0xbc, 0xe3, 0x97, 0x72, 0xda, 0x64, 0xa8, 0xc2, 0x4b, 0x7, 0xe5, 0xaf, 0xb5, 0x7, 0x1b, 0x48, 0xcd, 0x7f, 0xe6, 0x1f, 0xd1, 0x1c, 0xc5, 0x53, 0xb6, 0x47, 0x53, 0x5, 0x35, 0x95, 0x33, 0xf5, 0x4f, 0x1c, 0x52, 0xa4, 0xe6, 0xec, 0x85},
			ProposerID:         flow.Identifier{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			ProposerSigData:    []byte{0xa8, 0xe6, 0x23, 0x2e, 0x38, 0x6a, 0xb4, 0xed, 0xd1, 0x51, 0x24, 0xfd, 0xd5, 0x58, 0xee, 0x3f, 0x99, 0x74, 0xdf, 0xdf, 0x12, 0x1a, 0x96, 0x99, 0xd7, 0x22, 0xe4, 0xcc, 0x94, 0x30, 0xae, 0xc1, 0x5, 0xce, 0xd4, 0xa6, 0x54, 0xb6, 0xe5, 0x3c, 0xb5, 0x39, 0x44, 0x9d, 0x5d, 0x89, 0xa8, 0x66, 0x8f, 0x17, 0x23, 0xf4, 0xae, 0xce, 0xf9, 0xc1, 0x5c, 0x15, 0x46, 0x5c, 0x87, 0x1b, 0xf6, 0xa9, 0x5a, 0xf7, 0xa9, 0xa9, 0x84, 0xfe, 0x7d, 0xc3, 0xc2, 0x12, 0xe9, 0xd8, 0xa3, 0xe3, 0x87, 0x8f, 0x7f, 0x97, 0x41, 0x5e, 0x32, 0xb3, 0x83, 0x55, 0x29, 0xb, 0xda, 0x1, 0x19, 0x1a, 0xac, 0xee},
		}

	case 41:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0x7f, 0x10, 0x5b, 0x61, 0x34, 0x87, 0xeb, 0x43, 0x7c, 0x28, 0xe3, 0xbd, 0x39, 0x9a, 0x5, 0xc8, 0x11, 0x58, 0x33, 0x3d, 0xb8, 0x53, 0xc0, 0xec, 0x91, 0x63, 0x4, 0xd6, 0x69, 0x7c, 0x4c, 0x17},
			Height:      41,
			PayloadHash: flow.Identifier{0x1f, 0xf0, 0x59, 0xc6, 0xb0, 0xde, 0xcd, 0x6a, 0xec, 0x7f, 0x87, 0x28, 0xcf, 0x18, 0x92, 0xd1, 0xf, 0x7c, 0x37, 0xa6, 0x3d, 0xd0, 0x9c, 0xf3, 0xb9, 0x6, 0xb9, 0x4e, 0xab, 0x8b, 0xf8, 0xc},
			Timestamp:   time.Unix(0, 1632143326915586100).UTC(),
			View:        42,
			ParentVoterIDs: []flow.Identifier{
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			},
			ParentVoterSigData: []byte{0xb1, 0x1c, 0xd7, 0x88, 0xc2, 0xb0, 0x52, 0x87, 0x69, 0xc4, 0x6e, 0x66, 0x66, 0x1c, 0xce, 0x83, 0xa2, 0x44, 0x97, 0x7f, 0xe7, 0xac, 0xf8, 0xb9, 0xeb, 0xbf, 0x24, 0x86, 0x27, 0xbb, 0x83, 0xc3, 0x2, 0x13, 0x23, 0x88, 0x24, 0x1a, 0xae, 0x61, 0xf6, 0xeb, 0x3f, 0x2d, 0x42, 0x38, 0x9, 0xa6, 0xaf, 0xba, 0xc1, 0x3c, 0x7e, 0xfb, 0xfd, 0x3, 0x7e, 0x49, 0x96, 0x4d, 0x8f, 0x69, 0xc, 0xed, 0x40, 0x1f, 0xe2, 0x53, 0x31, 0x37, 0xd5, 0x96, 0x70, 0xcc, 0x3d, 0xfd, 0xa8, 0x79, 0x6f, 0x71, 0x3d, 0xcb, 0x92, 0xb0, 0x40, 0xe, 0x12, 0x17, 0xd9, 0x22, 0x3e, 0x4d, 0x21, 0x14, 0x15, 0x78},
			ProposerID:         flow.Identifier{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			ProposerSigData:    []byte{0xa6, 0xa0, 0x1, 0x72, 0xa, 0x9, 0x52, 0x2f, 0xa7, 0xd4, 0xa1, 0x66, 0x95, 0x46, 0x9b, 0x7f, 0x88, 0xc3, 0xc4, 0x5b, 0xa2, 0xf1, 0xa, 0x96, 0xf3, 0x8d, 0xca, 0x12, 0xa3, 0xa, 0x3e, 0x1f, 0x6c, 0x64, 0xb6, 0x95, 0x61, 0xda, 0x9c, 0xa, 0x18, 0x6b, 0xe0, 0x66, 0xd, 0x48, 0x92, 0x61, 0x81, 0x29, 0xfb, 0xfb, 0x8a, 0x66, 0x98, 0xf9, 0xec, 0x29, 0x6e, 0xb4, 0xe3, 0x43, 0xff, 0x19, 0x2, 0xa5, 0xa4, 0xc3, 0x69, 0x40, 0x7d, 0x2e, 0x20, 0xfc, 0xc6, 0xd, 0xa5, 0xad, 0x51, 0xa0, 0x2d, 0x92, 0xec, 0xf0, 0xef, 0xfa, 0x58, 0x94, 0x1a, 0xf5, 0xa3, 0x23, 0x64, 0x1f, 0x7, 0x1c},
		}

	case 47:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0x6d, 0x4e, 0xd3, 0xb4, 0x4e, 0xa6, 0xe6, 0xed, 0x12, 0xd7, 0x6c, 0x9b, 0x7a, 0xb1, 0x25, 0x39, 0x1a, 0x89, 0x35, 0x3f, 0x94, 0x5d, 0x18, 0xb7, 0x27, 0x6e, 0x15, 0x24, 0x27, 0x13, 0x4d, 0xd2},
			Height:      47,
			PayloadHash: flow.Identifier{0x2d, 0xa7, 0x24, 0x32, 0x51, 0x14, 0x2d, 0x7e, 0xce, 0x10, 0x80, 0xcb, 0xdd, 0x8a, 0xf2, 0x71, 0x31, 0xf2, 0xd0, 0xd0, 0xbf, 0xa6, 0xbc, 0x9, 0x3a, 0x7c, 0x91, 0xf4, 0xf8, 0x34, 0xd5, 0xbb},
			Timestamp:   time.Unix(0, 1632143332710441700).UTC(),
			View:        48,
			ParentVoterIDs: []flow.Identifier{
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			},
			ParentVoterSigData: []byte{0x98, 0x63, 0xf8, 0x97, 0x9e, 0x5b, 0x72, 0xf0, 0x7f, 0xca, 0xb5, 0xb5, 0x1, 0x31, 0x44, 0x86, 0x9a, 0xa0, 0x3d, 0xd0, 0x25, 0x9f, 0x40, 0x5, 0x93, 0x43, 0xde, 0x2f, 0x15, 0x8f, 0xbc, 0x3, 0x11, 0xcd, 0x43, 0x5c, 0x42, 0x8f, 0x1d, 0xea, 0x30, 0xde, 0xda, 0xd1, 0x5f, 0x35, 0xe0, 0xc1, 0xb1, 0x74, 0xe6, 0xb0, 0xf1, 0xc3, 0x19, 0xc8, 0x29, 0x6a, 0xf9, 0xeb, 0x5a, 0x8e, 0x98, 0x1c, 0xf6, 0x18, 0x0, 0xbb, 0xee, 0xfa, 0x1f, 0x6, 0x21, 0x5, 0x42, 0x1, 0xe4, 0x9, 0x6c, 0xf9, 0x50, 0xca, 0x68, 0x7d, 0xb0, 0x99, 0xa0, 0xaa, 0x77, 0x8a, 0xe8, 0x95, 0xc2, 0xb7, 0x94, 0xc8},
			ProposerID:         flow.Identifier{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			ProposerSigData:    []byte{0x91, 0x6d, 0x29, 0x7, 0xd1, 0xa5, 0x18, 0x82, 0x5e, 0xcf, 0x73, 0x17, 0x9a, 0xf4, 0xd2, 0x31, 0x1f, 0xd4, 0xea, 0x8b, 0x93, 0x37, 0x4a, 0x31, 0x5, 0xf8, 0x27, 0x83, 0x15, 0xed, 0xf4, 0x16, 0xe7, 0xb4, 0x2b, 0x54, 0x26, 0xb5, 0xe9, 0x93, 0xb5, 0xe4, 0x75, 0x8c, 0x56, 0x99, 0xa, 0xe4, 0xb1, 0x44, 0x66, 0x77, 0x85, 0x6, 0xb6, 0x15, 0xf6, 0x80, 0x2d, 0x65, 0xc, 0x3b, 0xf9, 0xac, 0xa3, 0xbc, 0x34, 0xd6, 0x73, 0x5e, 0x21, 0x20, 0x40, 0xf4, 0x9d, 0xf3, 0xcb, 0xcb, 0xbf, 0xa0, 0x8b, 0x74, 0x38, 0x4, 0x7, 0x77, 0xf1, 0x27, 0xe1, 0xf5, 0x3f, 0x7b, 0x11, 0xab, 0x6f, 0xd5},
		}

	case 57:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0x83, 0x1b, 0x13, 0x7, 0x45, 0xcd, 0x9c, 0xaf, 0xc2, 0x93, 0xe6, 0x5e, 0xc9, 0x6, 0x54, 0x24, 0x2b, 0xd1, 0xee, 0xfe, 0xb1, 0x35, 0x2c, 0xe5, 0x24, 0xa8, 0x59, 0xc7, 0xe2, 0xa, 0xf0, 0xa1},
			Height:      57,
			PayloadHash: flow.Identifier{0xbe, 0xf, 0xa4, 0x5a, 0x9f, 0x3a, 0xfc, 0xe8, 0x86, 0x1d, 0x6, 0x5c, 0xd, 0xf1, 0xcb, 0xa9, 0xeb, 0xf8, 0x8c, 0x77, 0x9c, 0xee, 0x9b, 0x38, 0x60, 0x4c, 0x63, 0x48, 0x93, 0xfe, 0x82, 0x1},
			Timestamp:   time.Unix(0, 1632143342380069100).UTC(),
			View:        58,
			ParentVoterIDs: []flow.Identifier{
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			},
			ParentVoterSigData: []byte{0x82, 0xe2, 0x43, 0x48, 0x18, 0x4e, 0x6a, 0x75, 0x60, 0x3, 0xe8, 0x1c, 0xb3, 0x5a, 0x81, 0x14, 0x25, 0xbd, 0xfd, 0xaf, 0xaa, 0x78, 0x9c, 0x41, 0x1, 0xd7, 0xdc, 0x12, 0xf4, 0xbc, 0x72, 0x60, 0xd2, 0xf5, 0xdc, 0xc1, 0xfc, 0x80, 0x29, 0xe6, 0xe1, 0xd7, 0xd5, 0x47, 0xe9, 0x40, 0xc1, 0xdd, 0xb5, 0x86, 0xb7, 0x29, 0xa7, 0x78, 0x5c, 0x84, 0x1d, 0xa5, 0xa9, 0xef, 0x36, 0x7f, 0xc4, 0x82, 0xad, 0x11, 0x8f, 0x61, 0x62, 0x89, 0xcc, 0xfc, 0xcd, 0xcd, 0xa9, 0x86, 0x1, 0x29, 0x64, 0x31, 0xd9, 0xe7, 0x95, 0xee, 0x5e, 0x0, 0x6, 0x2b, 0x69, 0x24, 0x9, 0xa1, 0xa8, 0xe8, 0x6e, 0x7d},
			ProposerID:         flow.Identifier{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			ProposerSigData:    []byte{0xa7, 0xff, 0x48, 0xcc, 0x68, 0xe, 0xb1, 0xe5, 0xbf, 0xc5, 0x58, 0xf2, 0xaa, 0xc7, 0xa1, 0x7, 0x99, 0x65, 0xfb, 0xe5, 0xe4, 0x94, 0x26, 0xef, 0x8b, 0x5b, 0xa, 0xe, 0x95, 0x38, 0xb6, 0xd5, 0x53, 0xb9, 0xdd, 0x63, 0xab, 0x91, 0x81, 0xd6, 0xe4, 0x17, 0xaf, 0x26, 0x3b, 0x21, 0xcf, 0x7c, 0x8d, 0x56, 0xdd, 0xb8, 0xce, 0x24, 0x40, 0xaa, 0xb1, 0x35, 0xbe, 0xb7, 0xe, 0x25, 0x41, 0x3, 0x14, 0x48, 0xfe, 0x42, 0xb2, 0x35, 0x70, 0x0, 0x4d, 0x8f, 0xb0, 0x5a, 0xd7, 0x1d, 0x17, 0xe4, 0xda, 0xeb, 0xf6, 0x83, 0x0, 0x4f, 0x1c, 0xf0, 0x1, 0xf6, 0xce, 0x60, 0x6a, 0xc, 0x1, 0xc},
		}

	case 60:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0x83, 0x28, 0x9a, 0xba, 0xed, 0xc0, 0xe6, 0xf8, 0x33, 0x34, 0xb7, 0xf1, 0x12, 0xaa, 0x3c, 0x93, 0xe9, 0x59, 0xad, 0x7c, 0x1f, 0x4, 0xf6, 0xf1, 0xa7, 0xc5, 0xa8, 0xe1, 0x14, 0xb5, 0xdf, 0x32},
			Height:      60,
			PayloadHash: flow.Identifier{0x9, 0xc1, 0xc9, 0xf3, 0x7b, 0xb, 0x7a, 0xc3, 0xe5, 0x44, 0xc6, 0x2e, 0xc6, 0xc0, 0xa0, 0x6e, 0xc2, 0x10, 0x69, 0x8d, 0x2c, 0xea, 0xc2, 0xf4, 0x65, 0xa5, 0x8b, 0xe6, 0x89, 0xbf, 0xf0, 0x1},
			Timestamp:   time.Unix(0, 1632143345279075600).UTC(),
			View:        61,
			ParentVoterIDs: []flow.Identifier{
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			},
			ParentVoterSigData: []byte{0x80, 0xcd, 0x25, 0x65, 0x66, 0x78, 0x1c, 0xa6, 0xf4, 0x33, 0x24, 0x69, 0xd6, 0x40, 0x92, 0x8b, 0xb5, 0x9d, 0xe0, 0x36, 0xfe, 0x79, 0x9c, 0xe9, 0xa3, 0x34, 0x1e, 0xc0, 0xe5, 0x74, 0xda, 0x4d, 0x6a, 0x29, 0x88, 0x15, 0x44, 0xf6, 0xdf, 0x74, 0xf9, 0x5, 0xfb, 0xc0, 0xe1, 0x38, 0x92, 0xa2, 0xad, 0xf1, 0x3d, 0xad, 0xfa, 0xb1, 0x52, 0x39, 0x18, 0x88, 0xc9, 0xb3, 0xd4, 0xa, 0x86, 0x45, 0xe9, 0x21, 0xe, 0x92, 0x47, 0xfb, 0x24, 0xbe, 0xa7, 0x74, 0xd3, 0xfe, 0xb5, 0xf8, 0x6, 0x59, 0x53, 0x4b, 0x51, 0x96, 0xae, 0x79, 0x17, 0x99, 0x50, 0xa8, 0x75, 0x92, 0xb0, 0xe2, 0xe0, 0xd7},
			ProposerID:         flow.Identifier{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			ProposerSigData:    []byte{0x95, 0xe0, 0x8a, 0x9c, 0xf6, 0xae, 0xc7, 0x5d, 0x4d, 0x3, 0x46, 0xcd, 0xed, 0x93, 0xbf, 0x72, 0xe8, 0x3c, 0x30, 0xa4, 0xfc, 0xd0, 0xd, 0x2, 0x9f, 0xa3, 0xae, 0x50, 0x24, 0xb2, 0xc5, 0x99, 0x0, 0x55, 0xd2, 0xbc, 0xaa, 0x72, 0x3a, 0x28, 0x29, 0x4a, 0x43, 0xad, 0xc6, 0x7b, 0xbf, 0xfd, 0xa4, 0xf0, 0xf0, 0x6a, 0xdd, 0x63, 0xe5, 0x1d, 0x95, 0x8c, 0x5a, 0xd5, 0x6e, 0x23, 0xdd, 0x8e, 0x2e, 0xf, 0x4d, 0xbc, 0x87, 0xa7, 0xb5, 0x95, 0xc2, 0x5a, 0xfe, 0x31, 0x43, 0x4d, 0x51, 0x90, 0xc7, 0xf3, 0x65, 0xc2, 0x6e, 0x9c, 0x9, 0x75, 0x3d, 0x1d, 0x50, 0x30, 0xd3, 0x66, 0x13, 0x8b},
		}

	case 65:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0x52, 0x87, 0x1c, 0xdf, 0xb7, 0xd0, 0x96, 0xb2, 0xc9, 0x85, 0x73, 0x31, 0x92, 0x7f, 0xe7, 0xd5, 0xe5, 0x69, 0x2b, 0xb3, 0x6e, 0x22, 0xcd, 0x9, 0x26, 0xe1, 0x6a, 0x42, 0x87, 0xe3, 0xd4, 0x22},
			Height:      65,
			PayloadHash: flow.Identifier{0x95, 0x81, 0xe2, 0x6f, 0xcf, 0x61, 0x55, 0x3e, 0xed, 0x14, 0xd9, 0x4d, 0x4f, 0x50, 0x22, 0xe, 0xf3, 0x56, 0xc5, 0xcb, 0x86, 0x1f, 0xd3, 0x96, 0xf2, 0x5b, 0xdc, 0xee, 0x8e, 0x98, 0xe5, 0xf8},
			Timestamp:   time.Unix(0, 1632143350194403000).UTC(),
			View:        66,
			ParentVoterIDs: []flow.Identifier{
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			},
			ParentVoterSigData: []byte{0xb4, 0x54, 0x76, 0x9d, 0x8a, 0xa4, 0xdd, 0xb, 0x48, 0xbd, 0x4e, 0xc7, 0xff, 0x87, 0xdc, 0x83, 0xa2, 0x17, 0xfb, 0xa6, 0x31, 0x1c, 0x8, 0x7d, 0x6f, 0x79, 0x5e, 0xae, 0x8c, 0xcf, 0xdb, 0xd6, 0x69, 0x9d, 0x43, 0x1, 0x4e, 0x1c, 0xbe, 0x9c, 0xcb, 0x74, 0x2a, 0x76, 0x2c, 0x6d, 0x57, 0xd4, 0xa5, 0x5c, 0x10, 0xff, 0xc9, 0x7b, 0x17, 0x39, 0xec, 0x28, 0x23, 0xac, 0x6b, 0x7a, 0x82, 0x5b, 0xb, 0x35, 0xa, 0x1, 0x4c, 0xb6, 0x60, 0x77, 0x87, 0x70, 0xf0, 0x55, 0xa8, 0x87, 0xfc, 0x4e, 0xc0, 0x7c, 0xe6, 0x75, 0xd1, 0xb8, 0xb5, 0xe9, 0x8a, 0xc4, 0x20, 0x90, 0xf1, 0x63, 0x6c, 0xbb},
			ProposerID:         flow.Identifier{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			ProposerSigData:    []byte{0xa2, 0xd4, 0xf4, 0x72, 0x50, 0x99, 0xe1, 0x5e, 0xf6, 0xf5, 0xab, 0x18, 0xf1, 0xd6, 0x22, 0x30, 0xc7, 0xe0, 0xbc, 0xb2, 0xbf, 0x3c, 0xe2, 0xff, 0x42, 0x51, 0xa, 0xf8, 0xc1, 0xac, 0xab, 0xea, 0xc5, 0x72, 0x71, 0x89, 0x68, 0xa, 0x93, 0x63, 0x10, 0x4b, 0xdf, 0x22, 0xd0, 0x74, 0x50, 0x5f, 0x8b, 0x42, 0xd2, 0xdb, 0x4a, 0x18, 0x67, 0x13, 0x9f, 0x9d, 0x1d, 0x9c, 0xff, 0xe8, 0xa1, 0xd0, 0xe5, 0x95, 0xfd, 0xdd, 0xa8, 0x3d, 0x90, 0xa1, 0xc9, 0x85, 0xdf, 0x6f, 0xe3, 0xf8, 0x9b, 0x90, 0xe6, 0x1c, 0x21, 0x24, 0x3e, 0x37, 0xdf, 0x21, 0x68, 0x4f, 0x66, 0xe2, 0xa4, 0xfd, 0x7c, 0x1e},
		}

	case 116:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0x7, 0x66, 0xe2, 0x11, 0xc8, 0xcb, 0x7, 0x9e, 0xc8, 0xdd, 0xc2, 0x41, 0x23, 0x31, 0xe0, 0x7b, 0xc7, 0xa3, 0x6, 0x46, 0xa3, 0xff, 0x26, 0x7e, 0x6f, 0xf2, 0xe2, 0xed, 0x84, 0x31, 0xed, 0xfe},
			Height:      116,
			PayloadHash: flow.Identifier{0xdf, 0x5, 0xac, 0x6c, 0x1e, 0xf8, 0x8d, 0xce, 0x1d, 0xee, 0xf3, 0x9e, 0x23, 0x73, 0xbb, 0x86, 0x9a, 0xac, 0x62, 0x66, 0x67, 0xde, 0xd6, 0x99, 0xc, 0xe4, 0x4, 0x78, 0x5, 0xd2, 0x71, 0xe3},
			Timestamp:   time.Unix(0, 1632143400616305000).UTC(),
			View:        117,
			ParentVoterIDs: []flow.Identifier{
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			},
			ParentVoterSigData: []byte{0x87, 0xda, 0xce, 0x4a, 0xfd, 0x53, 0xd6, 0x91, 0xe8, 0xb9, 0x40, 0x6, 0x88, 0xe2, 0x93, 0x4, 0x21, 0xed, 0x50, 0xd1, 0x10, 0xf3, 0x90, 0xb9, 0xae, 0x64, 0x58, 0x44, 0xca, 0x76, 0xca, 0x9a, 0x3f, 0xd2, 0x0, 0xa3, 0x1b, 0x39, 0xf1, 0xe9, 0x5a, 0xaa, 0x5d, 0xcb, 0xbd, 0xbd, 0xd0, 0xda, 0x86, 0x8b, 0x84, 0x5a, 0x62, 0x9, 0x49, 0xd0, 0xfc, 0x8c, 0x73, 0x49, 0x3b, 0x9c, 0x54, 0xf3, 0x62, 0x69, 0xac, 0xb5, 0x8, 0xb8, 0xee, 0x16, 0x35, 0xa, 0x7c, 0xdf, 0xcf, 0xe8, 0x8d, 0xf6, 0xbd, 0xe1, 0xa6, 0xda, 0x13, 0x80, 0xbb, 0x63, 0xbc, 0xf9, 0x67, 0x2b, 0x56, 0x7a, 0x35, 0x6b},
			ProposerID:         flow.Identifier{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			ProposerSigData:    []byte{0x8b, 0xfe, 0x4, 0x7c, 0x81, 0x33, 0x83, 0xf, 0x3f, 0x62, 0x5f, 0xc2, 0xb2, 0x29, 0x96, 0x20, 0x72, 0x1, 0x99, 0xf0, 0x68, 0x23, 0x63, 0xce, 0x6a, 0x24, 0x0, 0x45, 0x3a, 0x2b, 0x72, 0xd, 0x94, 0xcd, 0xd9, 0xa0, 0x8e, 0x87, 0x2d, 0xa5, 0x56, 0x88, 0x7d, 0x4f, 0x14, 0x77, 0xa8, 0xe9, 0xb4, 0x9b, 0x4d, 0xe7, 0xdf, 0x8d, 0x34, 0x38, 0xfb, 0xbd, 0x35, 0xca, 0x26, 0xe0, 0xa9, 0x24, 0x68, 0xfd, 0x42, 0x2d, 0x3, 0x53, 0x54, 0xb1, 0xf2, 0xea, 0x4b, 0x76, 0xc1, 0x3e, 0x21, 0x8c, 0x9e, 0xb3, 0xbd, 0x46, 0x7b, 0xbf, 0xc, 0x9c, 0x6c, 0xd4, 0x6, 0x8c, 0x5a, 0xc7, 0x5b, 0x41},
		}

	case 164:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0xe6, 0x55, 0xb1, 0xd, 0xf3, 0xc5, 0x73, 0x4d, 0x54, 0xbb, 0xbe, 0x37, 0x54, 0xab, 0x37, 0x24, 0xdf, 0x29, 0x24, 0xa2, 0xc5, 0xf0, 0xe0, 0x60, 0xac, 0x56, 0x75, 0x64, 0x5a, 0xcd, 0x7f, 0x5d},
			Height:      164,
			PayloadHash: flow.Identifier{0x3a, 0xf7, 0xa0, 0x32, 0x4a, 0x22, 0xef, 0x42, 0x99, 0xf7, 0x77, 0xe8, 0x45, 0x96, 0xd7, 0x31, 0x4b, 0x8b, 0x15, 0x6b, 0x85, 0x3b, 0xea, 0x10, 0xcc, 0x24, 0x13, 0xa3, 0x2a, 0x45, 0x11, 0x25},
			Timestamp:   time.Unix(0, 1632143449328433900).UTC(),
			View:        165,
			ParentVoterIDs: []flow.Identifier{
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			},
			ParentVoterSigData: []byte{0xa6, 0x7b, 0x49, 0xd5, 0xc4, 0xa4, 0xfa, 0x5b, 0xcb, 0x56, 0xb6, 0xd1, 0x49, 0x40, 0xcf, 0xa0, 0xc6, 0x99, 0x21, 0x3f, 0x99, 0xd8, 0x36, 0x87, 0xc7, 0xce, 0xa0, 0x6d, 0x47, 0x36, 0x8d, 0xc7, 0x43, 0x1f, 0xde, 0x62, 0x6d, 0x63, 0xea, 0x5b, 0x9b, 0xf6, 0x40, 0x8f, 0xf0, 0x59, 0x46, 0xf2, 0xb1, 0x54, 0xa2, 0x2b, 0x5e, 0x37, 0xca, 0xbf, 0x98, 0xb1, 0x84, 0x68, 0x85, 0x7f, 0xfa, 0xad, 0x64, 0x37, 0x21, 0x8a, 0xfb, 0x71, 0x60, 0x22, 0xe1, 0xa6, 0x50, 0xe7, 0x38, 0xf2, 0x6f, 0xf0, 0x97, 0xb0, 0xfd, 0x70, 0xb3, 0xe, 0x45, 0xe3, 0x66, 0xf5, 0x8f, 0xda, 0x5e, 0x5, 0x3d, 0xd4},
			ProposerID:         flow.Identifier{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			ProposerSigData:    []byte{0x84, 0xe, 0xf2, 0xf5, 0x75, 0x83, 0x7b, 0x9d, 0xc5, 0xe0, 0x25, 0xee, 0x44, 0x46, 0x84, 0xbe, 0xad, 0x51, 0x2f, 0x4a, 0xf0, 0xac, 0xfa, 0x93, 0x1d, 0xa5, 0x43, 0xf2, 0xec, 0xdd, 0xa8, 0xe9, 0x19, 0x5a, 0xfa, 0x1b, 0xa0, 0x2, 0xe9, 0x19, 0x8c, 0x40, 0x1, 0x30, 0x5e, 0xf2, 0x7d, 0x1a, 0xb0, 0xb6, 0x48, 0x70, 0x27, 0x5b, 0xe6, 0xa4, 0x3d, 0xa9, 0x36, 0x84, 0x37, 0x4f, 0x4e, 0x68, 0x3a, 0xc7, 0xf1, 0x1b, 0xf9, 0xb5, 0x19, 0xdc, 0xcb, 0xe4, 0x7f, 0x84, 0xb9, 0x74, 0xde, 0xfe, 0xe5, 0x71, 0x9a, 0x5, 0x14, 0xa2, 0x81, 0xbd, 0x81, 0x61, 0xc4, 0x9e, 0xd, 0xd3, 0x45, 0x1a},
		}

	case 173:
		return flow.Header{
			ChainID:     dps.FlowLocalnet,
			ParentID:    flow.Identifier{0xad, 0xd7, 0x4e, 0x8a, 0xcc, 0x5d, 0x7c, 0x4a, 0xcc, 0xe9, 0xd8, 0x4b, 0x75, 0xe9, 0x96, 0xc5, 0xf6, 0xe6, 0x85, 0x53, 0x98, 0x87, 0xc0, 0x34, 0xbf, 0xde, 0x50, 0xbf, 0x9d, 0x55, 0x5d, 0xb9},
			Height:      173,
			PayloadHash: flow.Identifier{0xd2, 0xf8, 0x15, 0x32, 0x6, 0x5f, 0x26, 0xb1, 0xa5, 0x99, 0x42, 0xd5, 0xbe, 0x84, 0xb9, 0xb4, 0x75, 0x80, 0x66, 0x3, 0x85, 0x6b, 0x25, 0x88, 0xc, 0x75, 0x32, 0x57, 0x50, 0xad, 0x67, 0x64},
			Timestamp:   time.Unix(0, 1632143457961983100).UTC(),
			View:        174,
			ParentVoterIDs: []flow.Identifier{
				{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
				{0xa5, 0xa2, 0x6e, 0x4d, 0x4c, 0x45, 0x61, 0x49, 0xa4, 0x2d, 0xc6, 0x89, 0xcd, 0xf0, 0xf1, 0x6, 0x11, 0x36, 0x62, 0x78, 0x7, 0x7c, 0xf7, 0xa2, 0x74, 0x9c, 0x99, 0x96, 0x8d, 0xe0, 0xb, 0x73},
			},
			ParentVoterSigData: []byte{0x81, 0xa0, 0x55, 0x3b, 0xcc, 0xe3, 0x90, 0x1d, 0x1a, 0xed, 0x84, 0x32, 0xf2, 0xba, 0xd0, 0x4b, 0x2b, 0xb9, 0x23, 0xbf, 0x6f, 0x2b, 0x68, 0x66, 0x46, 0xce, 0xeb, 0xc8, 0x96, 0xbe, 0xc3, 0x53, 0xd9, 0xd3, 0x5c, 0x16, 0x22, 0x8e, 0x63, 0xd5, 0xcd, 0xa3, 0x85, 0x93, 0x51, 0xb0, 0xb, 0x70, 0xb4, 0x24, 0x63, 0x71, 0x3a, 0x94, 0xea, 0x70, 0x25, 0x98, 0x1d, 0xb3, 0xd6, 0xda, 0xd9, 0xba, 0x17, 0x31, 0x12, 0xee, 0x58, 0xf1, 0x42, 0x39, 0x13, 0xb2, 0xd4, 0x8e, 0x2c, 0xf7, 0x11, 0x69, 0xe, 0x45, 0xad, 0x40, 0x9c, 0xdf, 0x28, 0x92, 0x8c, 0x4c, 0x76, 0xbd, 0x97, 0xeb, 0x8e, 0x7f},
			ProposerID:         flow.Identifier{0x7c, 0xcb, 0xcd, 0x3, 0x92, 0x2c, 0xda, 0xa2, 0x78, 0x45, 0x33, 0x8f, 0xd0, 0xf9, 0x62, 0xc1, 0x25, 0x20, 0xce, 0x1c, 0x9a, 0x10, 0xa1, 0xd3, 0x84, 0xa4, 0x44, 0x74, 0x50, 0x75, 0xe0, 0x5a},
			ProposerSigData:    []byte{0xac, 0xae, 0x73, 0xd9, 0x93, 0x1a, 0xbe, 0x67, 0xa, 0x6a, 0x6e, 0x17, 0xfb, 0x1c, 0xfd, 0xaf, 0x32, 0x95, 0xe3, 0x37, 0xad, 0xd5, 0x1e, 0xa4, 0xd2, 0x86, 0x87, 0x9e, 0xc8, 0x49, 0xb6, 0x2f, 0xec, 0x18, 0x5e, 0x67, 0x71, 0xc7, 0x79, 0x9d, 0x1b, 0xa0, 0x67, 0x49, 0x13, 0x92, 0x55, 0x51, 0xb8, 0x1b, 0xe5, 0x2e, 0x61, 0xfa, 0xa6, 0xb8, 0x8a, 0xd1, 0x9f, 0xc6, 0xb6, 0x50, 0x75, 0x62, 0x79, 0xda, 0x14, 0x27, 0xcc, 0xbb, 0x3e, 0xf2, 0xb6, 0xa4, 0x9a, 0x95, 0xab, 0x40, 0x78, 0x40, 0x5d, 0x5f, 0x8d, 0xa, 0xb6, 0x11, 0x42, 0x0, 0x86, 0xbe, 0x26, 0xd5, 0x9a, 0xeb, 0x77, 0x6f},
		}

	default:
		return flow.Header{}
	}
}

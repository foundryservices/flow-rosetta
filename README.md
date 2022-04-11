# flow-rosetta
forked from [dapperlabs/flow-rosetta](https://github.com/dapperlabs/flow-rosetta)

## Description

The Flow Rosetta Server implements the Rosetta Data API specifications for the Flow network.
It uses the Flow DPS Server's GRPC API as the backend to query the required data.
Flow core contract addresses are derived from the chain ID with which the service is started.
This allows the Rosetta API to access state remotely, or locally by running the Flow DPS Server on the same host.

## Dependencies

Go `v1.16` or higher is required to compile `flow-dps-rosetta`.
Only Linux amd64 builds are supported, because of the dependency to the [`flow-go/crypto`](https://github.com/onflow/flow-go/tree/master/crypto) package.

In order to build the live binary, the following extra steps and dependencies are required:

* [`CMake`](https://cmake.org/install/)

Please note that the flow-go repository should be cloned into this projects parent directory with its default name, so that the Go module replace statement works as intended: `replace github.com/onflow/flow-go/crypto => ../flow-go/crypto`.

* `git clone git@github.com:onflow/flow-go.git`
* `cd flow-go/crypto`
* `git checkout v0.25.0`
* `go generate`

You can then verify that the installation of the flow-go crypto package has been successful by running the tests of the project.

## Testing

The Flow system model, where resources can be moved freely between accounts without generating events, makes it impossible to fully reconcile account balances on the Rosetta Data API.
As a consequence, balance reconciliation has to be disabled when running the Rosetta CLI against the Flow Rosetta Server.

Additionally, some of the events generated by Flow are not accurately reflecting the account address they relate to.
This is also due to the same resource-based smart contract programming model.
For instance, when an account receives vaults from different locations to execute a swap of tokens, the events related to this swap might indicate the swap
contract's address, as it uses volatile vaults.

Currently, the only way to work around this issue is to create exemptions for accounts which contain such smart contracts.
As account balances using non-standard approaches to transfer Flow tokens can already not be reconciled, this is an acceptable limitation.
In general, transactions by Rosetta should not be used to deduce account balances.
Full historical account balance lookup is available and should thus be prefered to determine the account balance at any block height.

The discussed configuration is available in the `flow.json` and `exemptions.json` files for the `mainnet-9` spork DPS.
The following command can be executed to validate the Data API for that spork:

```sh
rosetta-cli check:data --configuration-file flow.json
```

For VSCode add this to `.vscode/settings.json`
```json
{
  "gopls": {
    "env": {
      "GOFLAGS": "-tags=relic,integration"
    },
  },
}
```

## Usage

```sh
Usage of flow-rosetta-server:
  -c, --access-api string        host address for Flow network\'s Access API endpoint (default "access.mainnet.nodes.onflow.org:9000")
  -e, --cache uint               maximum cache size for register reads in bytes (default 1000000000)
  -a, --dps-api string           host address for GRPC API endpoint (default "127.0.0.1:5005")
      --dump-requests            print out full request and responses
  -l, --level string             log output level (default "info")
  -p, --port uint16              port to host Rosetta API on (default 8080)
      --smart-status-codes       enable smart non-500 HTTP status codes for Rosetta API errors
  -t, --transaction-limit uint   maximum amount of transactions to include in a block response (default 200)
  -w, --wait-for-index           wait for index to be available instead of quitting right away, useful when DPS Live index bootstraps
```

## Example

The following command line starts the Flow Rosetta server for a main network spork on port 8080.
It uses a local instance of the Flow DPS Server for access to the execution state.

```sh
./flow-rosetta-server -a "127.0.0.1:5005" -p 8080
```

## Running without DPS Example

Some features can be used without the dps so you only need an access node for the mainnet you wish to run. Mainnet 16 example:

```sh
./flow-rosetta-server -c "access-001.mainnet16.nodes.onflow.org:9000" -p 8080
```

## Architecture

The Rosetta API needs its own documentation because of the amount of components it has that interact with each other.
The main reason for its complexity is that it needs to interact with the Flow Virtual Machine (FVM) and to translate between the Flow and Rosetta application domains.

### Invoker

This component, given a Cadence script, can execute it at any given height and return the value produced by the script.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps-rosetta/service/invoker)

### Retriever

The retriever uses the other components to retrieve account balances, blocks and transactions.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps-rosetta/service/retriever)

### Scripts

The script package produces Cadence scripts with the correct imports and storage paths, depending on the configured Flow chain ID.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps-rosetta/service/scripts)

### Validator

The Validator component validates whether the given Rosetta identifiers are valid.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps-rosetta/service/validator)

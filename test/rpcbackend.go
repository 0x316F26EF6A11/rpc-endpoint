/*
 * Dummy RPC backend. Implements Ethereum JSON-RPC calls that the tests need.
 */
package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/flashbots/rpc-endpoint/rpctypes"
	"github.com/flashbots/rpc-endpoint/server"
)

var getBundleStatusByTransactionHash_Response = rpctypes.GetBundleStatusByTransactionHashResponse{
	TxHash: TestTx_BundleFailedTooManyTimes_Hash,
	Status: "FAILED_BUNDLE",
}

var MockBackendLastRawRequest *http.Request
var MockBackendLastJsonRpcRequest *rpctypes.JsonRpcRequest
var MockBackendLastJsonRpcRequestTimestamp time.Time

func handleRpcRequest(req *rpctypes.JsonRpcRequest) (result interface{}, err error) {
	MockBackendLastJsonRpcRequest = req

	switch req.Method {
	case "eth_getTransactionCount":
		return "0x22", nil
		// return hex.DecodeString("0x22")

	case "eth_call":
		return "0x12345", nil

	case "eth_getTransactionReceipt":
		if req.Params[0] == TestTx_BundleFailedTooManyTimes_Hash {
			return nil, nil
		} else if req.Params[0] == TestTx_MM2_Hash {
			return nil, nil
		}

	case "eth_sendRawTransaction":
		return "tx-hash1", nil

	case "eth_sendPrivateTransaction":
		param := req.Params[0].(map[string]interface{})
		if param["tx"] == TestTx_BundleFailedTooManyTimes_RawTx {
			return TestTx_BundleFailedTooManyTimes_Hash, nil
		} else {
			return "tx-hash2", nil
		}

	case "net_version":
		return "3", nil

	case "null":
		return nil, nil

	case "eth_getBundleStatusByTransactionHash":
		return getBundleStatusByTransactionHash_Response, nil

	}

	return "", fmt.Errorf("no RPC method handler implemented for %s", req.Method)
}

func RpcBackendHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	MockBackendLastRawRequest = req
	MockBackendLastJsonRpcRequestTimestamp = server.Now()

	log.Printf("%s %s %s\n", req.RemoteAddr, req.Method, req.URL)

	w.Header().Set("Content-Type", "application/json")
	testHeader := req.Header.Get("Test")
	w.Header().Set("Test", testHeader)

	returnError := func(id interface{}, msg string) {
		log.Println("returnError:", msg)
		res := rpctypes.JsonRpcResponse{
			Id: id,
			Error: &rpctypes.JsonRpcError{
				Code:    -32603,
				Message: msg,
			},
		}

		if err := json.NewEncoder(w).Encode(res); err != nil {
			log.Printf("error writing response 1: %v - data: %s", err, res)
		}
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		returnError(-1, fmt.Sprintf("failed to read request body: %v", err))
		return
	}

	// Parse JSON RPC
	jsonReq := new(rpctypes.JsonRpcRequest)
	if err = json.Unmarshal(body, &jsonReq); err != nil {
		returnError(-1, fmt.Sprintf("failed to parse JSON RPC request: %v", err))
		return
	}

	rawRes, err := handleRpcRequest(jsonReq)
	if err != nil {
		returnError(jsonReq.Id, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	resBytes, err := json.Marshal(rawRes)
	if err != nil {
		fmt.Println("error mashalling rawRes:", rawRes, err)
	}

	res := rpctypes.NewJsonRpcResponse(jsonReq.Id, resBytes)

	// Write to client request
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("error writing response 2: %v - data: %s", err, rawRes)
	}
}

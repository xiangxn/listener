package flashbots

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

type Error struct {
	Code    int
	Message string
}

type SendPrivateTransactionResponse struct {
	Error  `json:"error,omitempty"`
	Result string `json:"result,omitempty"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type ParamsPrivateTransaction struct {
	Tx             string `json:"tx,omitempty"`
	MaxBlockNumber string `json:"maxBlockNumber,omitempty"`
	Preferences    struct {
		Fast    bool `json:"fast,omitempty"`
		Privacy struct {
			Builders []string `json:"builders,omitempty"`
		} `json:"privacy,omitempty"`
	} `json:"preferences,omitempty"`
}

func newMessage(method string, paramsIn ...interface{}) (*jsonrpcMessage, error) {
	msg := &jsonrpcMessage{Version: "2.0", ID: []byte(`1`), Method: method}
	if paramsIn != nil { // prevent sending "params":null
		var err error
		if msg.Params, err = json.Marshal(paramsIn); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func signPayload(payload []byte, prvKey *ecdsa.PrivateKey, pubKey *common.Address) (string, error) {
	if prvKey == nil || pubKey == nil {
		return "", errors.New("private or public key is not set")
	}
	signature, err := crypto.Sign(
		accounts.TextHash([]byte(hexutil.Encode(crypto.Keccak256(payload)))),
		prvKey,
	)
	if err != nil {
		return "", errors.Wrap(err, "sign the payload")
	}

	return pubKey.Hex() + ":" + hexutil.Encode(signature), nil
}

func FlashbotRequest(ctx context.Context, prvKey *ecdsa.PrivateKey, pubKey *common.Address, url string, method string, params ...interface{}) ([]byte, error) {
	if url == "" {
		url = "https://relay.flashbots.net"
	}
	msg, err := newMessage(method, params...)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling flashbot tx params")
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, io.NopCloser(bytes.NewReader(payload)))
	if err != nil {
		return nil, errors.Wrap(err, "creatting flashbot request")
	}
	signedP, err := signPayload(payload, prvKey, pubKey)
	if err != nil {
		return nil, errors.Wrap(err, "signing flashbot request")
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Flashbots-Signature", signedP)

	mevHTTPClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := mevHTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "flashbot request")
	}

	if resp.StatusCode/100 != 2 {
		respDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return nil, errors.Errorf("bad response status %v", resp.Status)
		}
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return nil, errors.Errorf("bad response resp respDump:%v", string(respDump))
		}
		return nil, errors.Errorf("bad response resp respDump:%v reqDump:%v", string(respDump), string(reqDump))
	}

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading flashbot reply")
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "closing flashbot reply body")
	}

	return res, nil
}

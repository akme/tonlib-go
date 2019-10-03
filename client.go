package tonlib

//#cgo linux CFLAGS: -I./lib/linux
//#cgo darwin CFLAGS: -I./lib/darwin
//#cgo linux LDFLAGS: -L./lib/linux -ltonlibjson -ltonlibjson_private -ltonlibjson_static -ltonlib
//#cgo darwin LDFLAGS: -L./lib/darwin -ltonlibjson -ltonlibjson_private -ltonlibjson_static -ltonlib
//#include <stdlib.h>
//#include <./lib/tonlib_client_json.h>
import "C"

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"
	"unsafe"
)

// client is the Telegram TdLib client
type Client struct {
	client unsafe.Pointer
	config Config
	wallet *TonWallet
}

// Config holds tonlibParameters
type Config struct {
	Timeout float32
}

// NewClient Creates a new instance of TONLib.
func NewClient(tonCnf *TONInitRequest, config Config) (*Client, error) {
	rand.Seed(time.Now().UnixNano())

	client := Client{client: C.tonlib_client_json_create(), config: combineConfig(config)}
	resp, err := client.executeAsynchronously(tonCnf)
	if err != nil {
		return &client, err
	}
	if st, ok := resp.Data["@type"]; ok && st == "ok" {
		return &client, nil
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return &client, fmt.Errorf("Error ton client init. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}
	fmt.Println("Init ton client result: ", string(resp.Raw), err)
	return &client, fmt.Errorf("Error ton client init. ")
}

// Init TonWallet and set it as default TonWallet for client
func (client *Client) InitWallet(key *TONPrivateKey, password []byte) (err error) {
	st := struct {
		Type       string   `json:"@type"`
		PrivateKey InputKey `json:"private_key"`
	}{
		Type:       "wallet.init",
		PrivateKey: key.getInputKey(password),
	}
	resp, err := client.executeAsynchronously(st)
	if err != nil {
		return err
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return fmt.Errorf("Error ton client init. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}
	if st, ok := resp.Data["@type"]; ok && st == "ok" {
		client.wallet = new(TonWallet)
		client.wallet.client = client
		return nil
	}
	return fmt.Errorf("Error ton TonWallet init. ")
}

// get wallet address method
func (client *Client) WalletGetAddress(pubKey string) (*TONAccountAddress, error) {
	if client.wallet == nil {
		return nil, fmt.Errorf("You must init wallet before. ")
	}
	return client.wallet.getAddress(pubKey)
}

// get wallet address method
func (client *Client) WalletSendGRAMM2Address(key *TONPrivateKey, password []byte, fromAddress, toAddress string, amount string) error {
	if client.wallet == nil {
		return fmt.Errorf("You must init wallet before. ")
	}
	return client.wallet.sendGRAMM2Address(key, password, fromAddress, toAddress, amount)
}

// get HEX full address
func (client *Client) UnpackAccountAddress(address string) (*TONUnpackedAddress, error) {
	st := struct {
		Type           string `json:"@type"`
		AccountAddress string `json:"account_address"`
	}{
		Type:           "unpackAccountAddress",
		AccountAddress: address,
	}
	resp, err := client.executeAsynchronously(st)
	if err != nil {
		return nil, err
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return nil, fmt.Errorf("Error ton client init. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}

	addressSt := TONUnpackedAddress{}
	err = json.Unmarshal(resp.Raw, &addressSt)
	if err != nil {
		return nil, err
	}
	return &addressSt, nil
}

// get HEX full address
func (client *Client) PackAccountAddress(packedAddr *TONUnpackedAddress, address string) (string, error) {
	fmt.Println("addr: ", packedAddr.Addr, address)
	st := struct {
		Type           string             `json:"@type"`
		AccountAddress TONUnpackedAddress `json:"account_address"`
	}{
		Type:           "packAccountAddress",
		AccountAddress: *packedAddr,
	}
	resp, err := client.executeAsynchronously(st)
	if err != nil {
		return "", err
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return "", fmt.Errorf("Error ton client init. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}

	addressSt := struct {
		TONAccountAddress
		Type string `json:"@type"`
	}{}
	err = json.Unmarshal(resp.Raw, &addressSt)
	if err != nil {
		return "", err
	}
	return addressSt.TONAccountAddress.AccountAddress, nil
}

// take account state
func (client *Client) GetAccountState(address string) (state *TONAccountState, err error) {
	st := struct {
		Type           string            `json:"@type"`
		AccountAddress TONAccountAddress `json:"account_address"`
	}{
		Type: "raw.getAccountState",
		AccountAddress: TONAccountAddress{
			AccountAddress: address,
		},
	}
	resp, err := client.executeAsynchronously(st)
	if err != nil {
		return state, err
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return state, fmt.Errorf("Error ton get account sate. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}

	state = new(TONAccountState)
	err = json.Unmarshal(resp.Raw, state)
	if err != nil {
		return state, err
	}
	return state, nil
}

func (client *Client) updateSendLiteServerQuery(id, data string) (res *TONResult, err error) {
	queryID, err := strconv.Atoi(id)
	if err != nil {
		return
	}
	st := struct {
		Type  string `json:"@type"`
		Id    int32  `json:"id"`
		Bytes []byte `json:"bytes"`
	}{
		Type:  "onLiteServerQueryResult",
		Bytes: []byte(data),
		Id:    int32(queryID),
	}
	return client.executeAsynchronously(st)
}

// todo
func (client *Client) SendMessage(initialAddress, destinationAddress, data string) (res *TONResult, err error) {
	st := struct {
		Type                string            `json:"@type"`
		Destination         TONAccountAddress `json:"destination"`
		InitialAccountState string            `json:"initial_account_state"`
		Data                string            `json:"data"`
	}{
		Type: "raw.sendMessage",
		Data: data,
		Destination: TONAccountAddress{
			AccountAddress: destinationAddress,
		},
		InitialAccountState: initialAddress,
	}
	return client.executeAsynchronously(st)
}

//
func (client *Client) updateSendLiteServerError(id, data string) (res *TONResult, err error) {
	queryID, err := strconv.Atoi(id)
	if err != nil {
		return
	}
	st := struct {
		Type  string `json:"@type"`
		Id    int32  `json:"id"`
		Bytes []byte `json:"bytes"`
	}{
		Type:  "onLiteServerQueryError",
		Bytes: []byte(data),
		Id:    int32(queryID),
	}
	return client.executeAsynchronously(st)
}

func (client *Client) getLogStream(id, data string) (res *TONResult, err error) {
	queryID, err := strconv.Atoi(id)
	if err != nil {
		return
	}
	st := struct {
		Type  string `json:"@type"`
		Id    int32  `json:"id"`
		Bytes []byte `json:"bytes"`
	}{
		Type:  "getLogStream",
		Bytes: []byte(data),
		Id:    int32(queryID),
	}
	return client.executeAsynchronously(st)
}

//fetch address`s transactions
func (client *Client) GetAccountTransactions(address string, lt int, hash string) (txs *TONTransactionsResponse, err error) {
	st := struct {
		Type              string                `json:"@type"`
		AccountAddress    TONAccountAddress     `json:"account_address"`
		FromTransactionId InternalTransactionId `json:"from_transaction_id"`
	}{
		Type: "raw.getTransactions",
		AccountAddress: TONAccountAddress{
			AccountAddress: address,
		},
		FromTransactionId: InternalTransactionId{
			Lt:   lt,
			Hash: hash,
		},
	}
	resp, err := client.executeAsynchronously(st)
	if err != nil {
		return txs, err
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return txs, fmt.Errorf("Error ton get account transactions. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}

	txs = new(TONTransactionsResponse)
	err = json.Unmarshal(resp.Raw, txs)
	return txs, err
}

// create privateKey
func (client *Client) CreatePrivateKey(password []byte) (key *TONPrivateKey, err error) {
	st := struct {
		Type             string `json:"@type"`
		LocalPassword    string `json:"local_password"`
		MnemonicPassword string `json:"mnemonic_password"`
		RandomExtraSeed  string `json:"random_extra_seed"`
	}{
		Type:             "createNewKey",
		LocalPassword:    base64.StdEncoding.EncodeToString(password),
		MnemonicPassword: base64.StdEncoding.EncodeToString([]byte(" " + "test mnemonic")),
	}
	resp, err := client.executeAsynchronously(st)
	if err != nil {
		return key, err
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return key, fmt.Errorf("Error ton create private key. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}

	key = new(TONPrivateKey)
	err = json.Unmarshal(resp.Raw, key)
	return key, err
}

//change localPassword
func (client *Client) ChangeLocalPassword(key *TONPrivateKey, password, newPassword []byte) (*TONPrivateKey, error) {
	st := struct {
		Type             string   `json:"@type"`
		NewLocalPassword string   `json:"new_local_password"`
		InputKey         InputKey `json:"input_key"`
	}{
		Type:             "changeLocalPassword",
		NewLocalPassword: base64.StdEncoding.EncodeToString(password),
		InputKey:         key.getInputKey(password),
	}
	resp, err := client.executeAsynchronously(st)
	if err != nil {
		return key, err
	}
	if st, ok := resp.Data["@type"]; ok && st == "error" {
		return key, fmt.Errorf("Error ton change key passowrd. Code %v. Message %s. ", resp.Data["code"], resp.Data["message"])
	}
	key = new(TONPrivateKey)
	err = json.Unmarshal(resp.Raw, key)
	return key, err
}

func (client *Client) Destroy() {
	C.tonlib_client_json_destroy(client.client)
}

/**
execute ton-lib asynchronously
*/
func (client *Client) executeAsynchronously(data interface{}) (*TONResult, error) {
	req, err := json.Marshal(data)
	if err != nil {
		return &TONResult{}, err
	}
	cs := C.CString(string(req))
	defer C.free(unsafe.Pointer(cs))

	C.tonlib_client_json_send(client.client, cs)
	result := C.tonlib_client_json_receive(client.client, DEFAULT_TIMEOUT)

	for result == nil {
		time.Sleep(1 * time.Second)
		result = C.tonlib_client_json_receive(client.client, DEFAULT_TIMEOUT)
	}

	var updateData TONResponse
	res := C.GoString(result)
	resB := []byte(res)
	err = json.Unmarshal(resB, &updateData)
	fmt.Println("fetch data: ", string(resB))
	if st, ok := updateData["@type"]; ok && st == "updateSendLiteServerQuery" {
		err = json.Unmarshal(resB, &updateData)
		updateReq := map[string]string{}
		err = json.Unmarshal(resB, &updateReq)
		if err == nil {
			client.updateSendLiteServerQuery(updateReq["id"], updateReq["data"])
		}
	}
	return &TONResult{Data: updateData, Raw: resB}, err
}

/**
execute ton-lib synchronously
*/
func (client *Client) executeSynchronously(data interface{}) (*TONResult, error) {
	req, _ := json.Marshal(data)
	cs := C.CString(string(req))
	defer C.free(unsafe.Pointer(cs))
	result := C.tonlib_client_json_execute(client.client, cs)

	var updateData TONResponse
	res := C.GoString(result)
	resB := []byte(res)
	err := json.Unmarshal(resB, &updateData)
	return &TONResult{Data: updateData, Raw: resB}, err
}

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/pkg/errors"
)

const OK = "OK"
const ADMIN_LIST_KEY = "ADMIN_LIST_KEY"
const DEFAULT_KEY = "DEFAULT"
const POSTFIX_AUTH_LIST = "|AUTH_LIST_KEY"
const POSTFIX_SECRET = "|SECRET_KEY"

type SecretKeeper struct {
	contractapi.Contract
}

type AuthSet struct {
	Pubkey map[string]struct{}
}

func (a *AuthSet) ExportToString() string {
	authlist := []string{}
	for key := range a.Pubkey {
		authlist = append(authlist, key)
	}
	sort.Strings(authlist)
	return strings.Join(authlist, "|")
}

func (a *AuthSet) ImportFromString(pubkeys string) error {
	pubkeylist := strings.Split(pubkeys, "|")
	for _, pubkey := range pubkeylist {
		a.Pubkey[pubkey] = struct{}{}
	}
	return nil
}

type Secret struct {
	Value string `json:Value`
}

func (t *SecretKeeper) InitSecretKeeperExt(ctx contractapi.TransactionContextInterface) error {
	// init adminSet
	adminSet := make(map[string]struct{})
	adminSet["Alice"] = struct{}{}
	adminSet["Bob"] = struct{}{}
	authSet := AuthSet{
		Pubkey: adminSet,
	}
	authSetJson, err := json.Marshal(authSet)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(ADMIN_LIST_KEY, authSetJson)
	if err != nil {
		return fmt.Errorf("failed to put %s to world state. %v", ADMIN_LIST_KEY, err)
	}

	return createSecret(ctx, authSet.ExportToString(), DEFAULT_KEY, "DefaultSecret")
}

func (t *SecretKeeper) AddAdmin(ctx contractapi.TransactionContextInterface, sig string, pubkey string) error {
	// check if the user allow to update admin list.
	valid, err := VerifySig(ctx, sig, ADMIN_LIST_KEY)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("VerifySig failed, User are not Admin")
	}

	// update the value
	return updateAuth(ctx, pubkey, ADMIN_LIST_KEY, true)
}

func (t *SecretKeeper) RemoveAdmin(ctx contractapi.TransactionContextInterface, sig string, pubkey string) error {
	// check if the user allow to update admin list
	valid, err := VerifySig(ctx, sig, ADMIN_LIST_KEY)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("VerifySig failed, User are not Admin")
	}

	// update the value
	return updateAuth(ctx, pubkey, ADMIN_LIST_KEY, false)
}

func (t *SecretKeeper) CreateSecret(ctx contractapi.TransactionContextInterface, sig, pubkeys, key, value string) error {
	// check if the user allow to create secret
	valid, err := VerifySig(ctx, sig, ADMIN_LIST_KEY)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("VerifySig failed, User are not Admin")
	}

	// check if the authKey and secretKey are not created
	authKey := key + POSTFIX_AUTH_LIST
	secretKey := key + POSTFIX_SECRET

	authJson, err := ctx.GetStub().GetState(authKey)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if authJson != nil {
		return fmt.Errorf("the secret %s already existed", key)
	}
	secretJson, err := ctx.GetStub().GetState(secretKey)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if secretJson != nil {
		return fmt.Errorf("the secret %s already existed", key)
	}
	return createSecret(ctx, pubkeys, key, value)
}

func (t *SecretKeeper) AddUser(ctx contractapi.TransactionContextInterface, sig string, pubkey string, key string) error {
	authKey := key + POSTFIX_AUTH_LIST

	// check if the user allow to update authSet
	valid, err := VerifySig(ctx, sig, authKey)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("VerifySig failed, User are not allowed to perform this action")
	}

	// update the value
	return updateAuth(ctx, pubkey, authKey, true)
}

func (t *SecretKeeper) RemoveUser(ctx contractapi.TransactionContextInterface, sig string, pubkey string, key string) error {
	authKey := key + POSTFIX_AUTH_LIST

	// check if the user allow to update authSet
	valid, err := VerifySig(ctx, sig, authKey)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("VerifySig failed, User are not allowed to perform this action")
	}

	// update the value
	return updateAuth(ctx, pubkey, authKey, false)
}

func (t *SecretKeeper) LockSecret(ctx contractapi.TransactionContextInterface, sig string, key string, value string) error {
	authKey := key + POSTFIX_AUTH_LIST
	secretKey := key + POSTFIX_SECRET

	// check if the user allow to update secret
	valid, err := VerifySig(ctx, sig, authKey)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("VerifySig failed, User are not allowed to perform this action")
	}

	// update the value
	return updateSecret(ctx, secretKey, value)
}

func (t *SecretKeeper) RevealSecret(ctx contractapi.TransactionContextInterface, sig string, key string) (*Secret, error) {
	authKey := key + POSTFIX_AUTH_LIST
	secretKey := key + POSTFIX_SECRET

	// check if the user allow to view the secret.
	valid, err := VerifySig(ctx, sig, authKey)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("VerifySig failed, User are not allowed to perform this action")
	}

	// reveal secret
	secretJson, err := ctx.GetStub().GetState(secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if secretJson == nil {
		return nil, fmt.Errorf("the asset %s does not exist", secretKey)
	}
	var secret Secret
	err = json.Unmarshal(secretJson, &secret)
	if err != nil {
		return nil, err
	}
	return &secret, nil
}

func createSecret(ctx contractapi.TransactionContextInterface, pubkeys, key, value string) error {
	authKey := key + POSTFIX_AUTH_LIST
	secretKey := key + POSTFIX_SECRET

	// extract public keys and save pubkeys
	pubkeySet := AuthSet{
		Pubkey: map[string]struct{}{},
	}
	pubkeySet.ImportFromString(pubkeys)
	pubkeySetJson, err := json.Marshal(pubkeySet)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(authKey, pubkeySetJson)
	if err != nil {
		return fmt.Errorf("failed to put %s to world state. %v", authKey, err)
	}

	// extract and save value
	return updateSecret(ctx, secretKey, value)
}

func updateAuth(ctx contractapi.TransactionContextInterface, pubkey string, key string, addElseDel bool) error {
	authSet, _ := GetAuthList(ctx, key)

	if addElseDel {
		authSet.Pubkey[pubkey] = struct{}{}
	} else {
		delete(authSet.Pubkey, pubkey)
	}

	authSetJson, err := json.Marshal(authSet)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(key, authSetJson)
	if err != nil {
		return fmt.Errorf("failed to put %s to world state. %v", key, err)
	}
	return nil
}

func updateSecret(ctx contractapi.TransactionContextInterface, key string, value string) error {
	newSecret := Secret{
		Value: value,
	}
	newSecretJson, err := json.Marshal(newSecret)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(key, newSecretJson)
	if err != nil {
		return fmt.Errorf("failed to put %s to world state. %v", key, err)
	}
	return nil
}

func GetAuthList(ctx contractapi.TransactionContextInterface, key string) (*AuthSet, error) {
	authSetJson, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if authSetJson == nil {
		return nil, fmt.Errorf("the asset %s does not exist", key)
	}

	var authSet AuthSet
	err = json.Unmarshal(authSetJson, &authSet)
	if err != nil {
		return nil, err
	}
	return &authSet, nil
}

func VerifySig(ctx contractapi.TransactionContextInterface, sig string, key string) (bool, error) {
	authSet, err := GetAuthList(ctx, key)
	if err != nil {
		return false, err
	}

	if _, exist := authSet.Pubkey[sig]; exist {
		return true, nil
	}

	return false, nil
}

package chaincode

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-private-chaincode/ercc/registry/fakes"
	"github.com/stretchr/testify/require"
)

func Setup(t *testing.T) (SecretKeeper, *fakes.ChaincodeStub, *fakes.TransactionContext, []byte, []byte, []byte) {
	chaincodeStub := &fakes.ChaincodeStub{}
	transactionContext := &fakes.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	secretKeeper := SecretKeeper{}
	err := secretKeeper.InitSecretKeeperExt(transactionContext)
	require.NoError(t, err)

	_, adminSetByte := chaincodeStub.PutStateArgsForCall(0)
	_, authSetByte := chaincodeStub.PutStateArgsForCall(1) // get default secret authlist
	_, secretByte := chaincodeStub.PutStateArgsForCall(2)  // get default secret value

	return secretKeeper, chaincodeStub, transactionContext, adminSetByte, authSetByte, secretByte
}

func TestWrongSignature(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, adminSetByte, _, _ := Setup(t)

	chaincodeStub.GetStateReturns(adminSetByte, nil)

	falseSig := "falseSignature"
	fakeSecret := "fakeSecret"
	fakeKey := "fakeKey"

	err := secretKeeper.AddAdmin(transactionContext, falseSig, falseSig)
	require.EqualError(t, err, "VerifySig failed, User are not Admin")

	err = secretKeeper.RemoveAdmin(transactionContext, falseSig, falseSig)
	require.EqualError(t, err, "VerifySig failed, User are not Admin")

	err = secretKeeper.CreateSecret(transactionContext, falseSig, falseSig, fakeKey, fakeSecret)
	require.EqualError(t, err, "VerifySig failed, User are not Admin")

	err = secretKeeper.AddUser(transactionContext, falseSig, falseSig, fakeKey)
	require.EqualError(t, err, "VerifySig failed, User are not allowed to perform this action")

	err = secretKeeper.RemoveUser(transactionContext, falseSig, falseSig, fakeKey)
	require.EqualError(t, err, "VerifySig failed, User are not allowed to perform this action")

	err = secretKeeper.LockSecret(transactionContext, falseSig, fakeKey, fakeSecret)
	require.EqualError(t, err, "VerifySig failed, User are not allowed to perform this action")

	secret, err := secretKeeper.RevealSecret(transactionContext, falseSig, fakeKey)
	require.EqualError(t, err, "VerifySig failed, User are not allowed to perform this action")
	require.Nil(t, secret)
}

func TestAddAdmin(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, adminSetByte, _, _ := Setup(t)
	chaincodeStub.GetStateReturns(adminSetByte, nil)

	aliceSig := "Alice"
	evePubKey := "Eve"

	// check if adminlist not contains eve
	var authSet AuthSet
	err := json.Unmarshal(adminSetByte, &authSet)
	require.NoError(t, err)
	_, exist := authSet.Pubkey[evePubKey]
	require.False(t, exist)

	err = secretKeeper.AddAdmin(transactionContext, aliceSig, evePubKey)
	require.NoError(t, err)

	// check if adminlist contains eve.
	_, authSetByte2 := chaincodeStub.PutStateArgsForCall(3)
	var authSet2 AuthSet
	err = json.Unmarshal(authSetByte2, &authSet2)
	require.NoError(t, err)
	_, exist = authSet2.Pubkey[evePubKey]
	require.True(t, exist)
}

func TestRemoveAdmin(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, adminSetByte, _, _ := Setup(t)
	chaincodeStub.GetStateReturns(adminSetByte, nil)

	aliceSig := "Alice"
	bobPubKey := "Bob"

	// check if adminlist contains bob.
	var authSet AuthSet
	err := json.Unmarshal(adminSetByte, &authSet)
	require.NoError(t, err)
	_, exist := authSet.Pubkey[bobPubKey]
	require.True(t, exist)

	err = secretKeeper.RemoveAdmin(transactionContext, aliceSig, bobPubKey)
	require.NoError(t, err)

	// check if adminlist doesn't contain bob anymore.
	_, authSetByte2 := chaincodeStub.PutStateArgsForCall(3)
	var authSet2 AuthSet
	err = json.Unmarshal(authSetByte2, &authSet2)
	require.NoError(t, err)
	_, exist = authSet2.Pubkey[bobPubKey]
	require.False(t, exist)
}

func TestAddUser(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, _, authSetByte, _ := Setup(t)
	chaincodeStub.GetStateReturns(authSetByte, nil)

	aliceSig := "Alice"
	evePubKey := "Eve"
	defaultKey := DEFAULT_KEY

	// check if authlist not contains eve
	var authSet AuthSet
	err := json.Unmarshal(authSetByte, &authSet)
	require.NoError(t, err)
	_, exist := authSet.Pubkey[evePubKey]
	require.False(t, exist)

	err = secretKeeper.AddUser(transactionContext, aliceSig, evePubKey, defaultKey)
	require.NoError(t, err)

	// check if authlist contains eve.
	_, authSetByte2 := chaincodeStub.PutStateArgsForCall(3)
	var authSet2 AuthSet
	err = json.Unmarshal(authSetByte2, &authSet2)
	require.NoError(t, err)
	_, exist = authSet2.Pubkey[evePubKey]
	require.True(t, exist)
}

func TestRemoveUser(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, _, authSetByte, _ := Setup(t)
	chaincodeStub.GetStateReturns(authSetByte, nil)

	aliceSig := "Alice"
	bobPubKey := "Bob"
	defaultKey := DEFAULT_KEY

	// check if authlist contains bob.
	var authSet AuthSet
	err := json.Unmarshal(authSetByte, &authSet)
	require.NoError(t, err)
	_, exist := authSet.Pubkey[bobPubKey]
	require.True(t, exist)

	err = secretKeeper.RemoveUser(transactionContext, aliceSig, bobPubKey, defaultKey)
	require.NoError(t, err)

	// check if authlist doesn't contain bob anymore.
	_, authSetByte2 := chaincodeStub.PutStateArgsForCall(3)
	var authSet2 AuthSet
	err = json.Unmarshal(authSetByte2, &authSet2)
	require.NoError(t, err)
	_, exist = authSet2.Pubkey[bobPubKey]
	require.False(t, exist)
}

func TestCreateSecret(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, adminSetByte, _, _ := Setup(t)
	chaincodeStub.GetStateReturnsOnCall(0, adminSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(1, nil, nil)
	chaincodeStub.GetStateReturnsOnCall(2, nil, nil)

	aliceSig := "Alice"
	newSecretKey := "newSecret"
	newSecretValue := "newValue"
	pubkeys := &AuthSet{
		Pubkey: map[string]struct{}{},
	}
	pubkeys.Pubkey["Alice"] = struct{}{}
	pubkeys.Pubkey["Bob"] = struct{}{}

	// first time will create success
	err := secretKeeper.CreateSecret(transactionContext, aliceSig, pubkeys.ExportToString(), newSecretKey, newSecretValue)
	require.NoError(t, err)

	_, authSetByte2 := chaincodeStub.PutStateArgsForCall(3)
	_, secretByte2 := chaincodeStub.PutStateArgsForCall(4)

	// check value
	var recvAuth AuthSet
	err = json.Unmarshal(authSetByte2, &recvAuth)
	require.NoError(t, err)
	require.Equal(t, recvAuth.ExportToString(), pubkeys.ExportToString())

	var recvSecretValue Secret
	err = json.Unmarshal(secretByte2, &recvSecretValue)
	require.NoError(t, err)

	// second time will create failed
	chaincodeStub.GetStateReturnsOnCall(3, authSetByte2, nil)
	chaincodeStub.GetStateReturnsOnCall(4, secretByte2, nil)
	err = secretKeeper.CreateSecret(transactionContext, aliceSig, pubkeys.ExportToString(), newSecretKey, newSecretValue)
	require.EqualError(t, err, fmt.Sprintf("the secret %s already existed", newSecretKey))

}

func TestLockSecret(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, _, authSetByte, _ := Setup(t)
	chaincodeStub.GetStateReturns(authSetByte, nil)

	aliceSig := "Alice"
	defaultKey := DEFAULT_KEY
	newSecret := "newSecret"

	err := secretKeeper.LockSecret(transactionContext, aliceSig, defaultKey, newSecret)
	require.NoError(t, err)

	// check secret key value.
	_, secretByte := chaincodeStub.PutStateArgsForCall(3)
	var secret Secret
	err = json.Unmarshal(secretByte, &secret)
	require.NoError(t, err)
	require.EqualValues(t, secret.Value, newSecret)
}

func TestRevealSecret(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, _, authSetByte, defaultSecretByte := Setup(t)
	chaincodeStub.GetStateReturns(authSetByte, nil)

	aliceSig := "Alice"
	defaultKey := DEFAULT_KEY

	var defaultSecret Secret
	err := json.Unmarshal(defaultSecretByte, &defaultSecret)
	require.NoError(t, err)

	// check the return value equal with the secret in test.
	chaincodeStub.GetStateReturnsOnCall(0, authSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(1, defaultSecretByte, nil)
	secret, err := secretKeeper.RevealSecret(transactionContext, aliceSig, defaultKey)
	require.NoError(t, err)
	require.EqualValues(t, secret.Value, defaultSecret.Value)
}

func TestNormalBehavior(t *testing.T) {
	secretKeeper, chaincodeStub, transactionContext, adminSetByte, _, _ := Setup(t)

	aliceSig := "Alice"
	bobSig := "Bob"
	eveSig := "Eve"

	// add eve to admin
	chaincodeStub.GetStateReturnsOnCall(0, adminSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(1, adminSetByte, nil)
	err := secretKeeper.AddAdmin(transactionContext, aliceSig, eveSig)
	require.NoError(t, err)
	_, adminSetByte = chaincodeStub.PutStateArgsForCall(3)

	// eve create newsecret
	newSecretKey := "newSecretKey"
	newSecretValue := "newSecretValue~v1"
	chaincodeStub.GetStateReturnsOnCall(2, adminSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(3, nil, nil)
	chaincodeStub.GetStateReturnsOnCall(4, nil, nil)
	err = secretKeeper.CreateSecret(transactionContext, eveSig, bobSig+"|"+eveSig, newSecretKey, newSecretValue)
	require.NoError(t, err)
	_, authSetByte := chaincodeStub.PutStateArgsForCall(4)
	_, secretByte := chaincodeStub.PutStateArgsForCall(5)

	// bob able to reveal newsecret
	chaincodeStub.GetStateReturnsOnCall(5, authSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(6, secretByte, nil)
	revSecret, err := secretKeeper.RevealSecret(transactionContext, bobSig, newSecretKey)
	require.NoError(t, err)
	require.Equal(t, revSecret.Value, newSecretValue)

	// bob add alice to newsecret
	chaincodeStub.GetStateReturnsOnCall(7, authSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(8, authSetByte, nil)
	err = secretKeeper.AddUser(transactionContext, bobSig, aliceSig, newSecretKey)
	require.NoError(t, err)
	_, authSetByte = chaincodeStub.PutStateArgsForCall(6)

	// alice remove bob from newsecret
	chaincodeStub.GetStateReturnsOnCall(9, authSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(10, authSetByte, nil)
	err = secretKeeper.RemoveUser(transactionContext, aliceSig, bobSig, newSecretKey)
	require.NoError(t, err)
	_, authSetByte = chaincodeStub.PutStateArgsForCall(7)

	// eve lock newsecret
	newSecretValue = "newSecretValue~v2"
	chaincodeStub.GetStateReturnsOnCall(11, authSetByte, nil)
	err = secretKeeper.LockSecret(transactionContext, eveSig, newSecretKey, newSecretValue)
	require.NoError(t, err)
	_, secretByte = chaincodeStub.PutStateArgsForCall(8)

	// bob not able to reveal newsecret
	chaincodeStub.GetStateReturnsOnCall(12, authSetByte, nil)
	chaincodeStub.GetStateReturnsOnCall(13, secretByte, nil)
	revSecret, err = secretKeeper.RevealSecret(transactionContext, bobSig, newSecretKey)
	require.EqualError(t, err, "VerifySig failed, User are not allowed to perform this action")
	require.Nil(t, revSecret)
}

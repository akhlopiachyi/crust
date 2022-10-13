package cored

import (
	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CoreumFoundation/coreum-tools/pkg/must"
)

// importMnemonicsToKeyring adds keys to local keystore
func importMnemonicsToKeyring(homeDir string, mnemonics map[string]string) {
	kr, err := keyring.New("cored", "test", homeDir, nil)
	must.OK(err)

	for name, mnemonic := range mnemonics {
		_, err := kr.NewAccount(name, mnemonic, "", sdk.GetConfig().GetFullBIP44Path(), hd.Secp256k1)
		must.OK(err)
	}
}

// PrivateKeyFromMnemonic generates private key from mnemonic
func PrivateKeyFromMnemonic(mnemonic string) (cosmossecp256k1.PrivKey, error) {
	kr := keyring.NewUnsafe(keyring.NewInMemory())

	_, err := kr.NewAccount("tmp", mnemonic, "", sdk.GetConfig().GetFullBIP44Path(), hd.Secp256k1)
	if err != nil {
		return cosmossecp256k1.PrivKey{}, err
	}

	privKeyHex, err := kr.UnsafeExportPrivKeyHex("tmp")
	if err != nil {
		panic(err)
	}

	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		panic(err)
	}
	return cosmossecp256k1.PrivKey{
		Key: privKeyBytes,
	}, nil
}

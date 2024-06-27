package master

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

var wiredKey *rsa.PrivateKey
var wiredPub *rsa.PublicKey

func loadWiredKeyPair() {
	pubFileName := "wired.pub"
	keyFileName := "wired.key"

	if _, err := os.Stat(pubFileName); os.IsNotExist(err) {
		fmt.Println("Public key not found")

		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			fmt.Println("Error generating private key:", err)
			os.Exit(1)
		}

		pubFile, err := os.Create(pubFileName)
		if err != nil {
			fmt.Println("Error creating public key file:", err)
			os.Exit(1)
		}
		defer pubFile.Close()

		if err := pem.Encode(pubFile, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey),
		}); err != nil {
			fmt.Println("Error encoding public key:", err)
			os.Exit(1)
		}

		//now write private key

		keyFile, err := os.Create(keyFileName)
		if err != nil {
			fmt.Println("Error creating private key file:", err)
			os.Exit(1)
		}
		defer keyFile.Close()

		privBytes := x509.MarshalPKCS1PrivateKey(priv)
		if err := pem.Encode(keyFile, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privBytes,
		}); err != nil {
			fmt.Println("Error encoding private key:", err)
			os.Exit(1)
		}

		wiredKey = priv
		wiredPub = &priv.PublicKey
		return

	}
	pubFile, err := os.Open(pubFileName)
	if err != nil {
		fmt.Println("Error opening public key file:", err)
		os.Exit(1)
	}

	pubBytes, err := io.ReadAll(pubFile)
	if err != nil {
		fmt.Println("Error reading public key file:", err)
		os.Exit(1)
	}

	block, _ := pem.Decode(pubBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		fmt.Println("Invalid Public key")
		os.Exit(1)
	}

	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		fmt.Println("Error parsing public key:", err)
		os.Exit(1)
	}

	wiredPub = pub

	keyFile, err := os.Open(keyFileName)
	if err != nil {
		fmt.Println("Error opening private key file:", err)
		os.Exit(1)
	}

	keyBytes, err := io.ReadAll(keyFile)
	if err != nil {
		fmt.Println("Error reading private key file:", err)
		os.Exit(1)
	}

	block, _ = pem.Decode(keyBytes)

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("Error parsing private key:", err)
		os.Exit(1)

	}

	wiredKey = priv
}

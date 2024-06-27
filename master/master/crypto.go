package master

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log"
	"os"
)

var wiredKey *rsa.PrivateKey
var wiredPub *rsa.PublicKey

func loadWiredKeyPair() {
	pubFileName := "wired.pub"
	keyFileName := "wired.key"

	if _, err := os.Stat(pubFileName); os.IsNotExist(err) {
		log.Println("Generating new RSA key pair...")

		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatal("Error generating RSA key pair:", err)
		}

		pubFile, err := os.Create(pubFileName)
		if err != nil {
			log.Println("Error creating public key file:", err)
		}
		defer pubFile.Close()

		if err := pem.Encode(pubFile, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey),
		}); err != nil {
			log.Fatal("Error encoding public key:", err)
		}

		keyFile, err := os.Create(keyFileName)
		if err != nil {
			log.Fatal("Error creating private key file:", err)
		}
		defer keyFile.Close()

		privBytes := x509.MarshalPKCS1PrivateKey(priv)
		if err := pem.Encode(keyFile, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privBytes,
		}); err != nil {
			log.Fatal("Error encoding private key:", err)
		}

		wiredKey = priv
		wiredPub = &priv.PublicKey
		return

	}

	pubFile, err := os.Open(pubFileName)
	if err != nil {
		log.Fatal("Error opening public key file:", err)
	}

	pubBytes, err := io.ReadAll(pubFile)
	if err != nil {
		log.Fatal("Error reading public key file:", err)
	}

	block, _ := pem.Decode(pubBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		log.Fatal("Failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		log.Fatal("Error parsing public key:", err)
	}

	wiredPub = pub

	keyFile, err := os.Open(keyFileName)
	if err != nil {
		log.Fatal("Error opening private key file:", err)
	}

	keyBytes, err := io.ReadAll(keyFile)
	if err != nil {
		log.Fatal("Error reading private key file:", err)
	}

	block, _ = pem.Decode(keyBytes)

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal("Error parsing private key:", err)
	}

	log.Println("Loaded RSA key pair")
	wiredKey = priv
}

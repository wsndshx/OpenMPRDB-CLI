package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/gopenpgp/v2/helper"
)

// initializationKey 初始化本地密钥
func initializationKey() (err error) {
	// 生成密钥
	rsaKey, err := crypto.GenerateKey(" ", " ", "rsa", 2048)
	if err != nil {
		return
	}

	// 存储私钥
	private_key, _ := rsaKey.Armor()
	err = ioutil.WriteFile("rsa-priv.pem", []byte(private_key), 0644)
	if err != nil {
		return
	}

	// 存储公钥
	public_key, _ := rsaKey.GetArmoredPublicKey()
	err = ioutil.WriteFile("rsa-pub.pem", []byte(public_key), 0644)
	if err != nil {
		return
	}

	return
}

// GenerateSignedMessage
func GenerateSignedMessage(text string) (string, error) {
	// 生成签名
	PGPSignature, err := SignatureData(text)
	if err != nil {
		return "", errors.New("签名时发生错误: " + err.Error())
	}

	return fmt.Sprintf("-----BEGIN PGP SIGNED MESSAGE-----\nHash: SHA512\n\n%s\n%s", text, PGPSignature), nil
}

// SignatureData 对指定文本进行签名, 返回签名后的内容
func SignatureData(text string) (string, error) {
	privkey, err := ioutil.ReadFile("rsa-priv.pem")
	if err != nil {
		return "", errors.New("无法读取本地私钥: " + err.Error())
	}

	message := crypto.NewPlainMessage([]byte(text))

	privateKeyObj, err := crypto.NewKeyFromArmored(string(privkey))
	if err != nil {
		return "", err
	}

	signingKeyRing, err := crypto.NewKeyRing(privateKeyObj)
	if err != nil {
		return "", err
	}

	pgpSignature, err := signingKeyRing.SignDetached(message)
	miao, _ := pgpSignature.GetArmored()
	return miao, nil
}

// EncryptSignMessage 对指定文本进行加密并签名
func EncryptSignMessage(text string) (string, error) {
	privkey, err := ioutil.ReadFile("rsa-priv.pem")
	if err != nil {
		return "", errors.New("无法读取本地私钥: " + err.Error())
	}

	pubkey, err := ioutil.ReadFile("rsa-pub.pem")
	if err != nil {
		return "", errors.New("无法读取本地公钥: " + err.Error())
	}
	armor, err := helper.EncryptSignMessageArmored(string(pubkey), string(privkey), nil, text)
	if err != nil {
		return "", errors.New("生成签名错误: " + err.Error())
	}
	return armor, nil
}

// verifySignature 验证签名是否正确
func verifySignature(text string, signature string) bool {
	pubkey, err := ioutil.ReadFile("rsa-pub.pem")
	if err != nil {
		log.Fatalln(err)
		return false
	}

	message := crypto.NewPlainMessage([]byte(text))
	pgpSignature, err := crypto.NewPGPSignatureFromArmored(signature)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	publicKeyObj, err := crypto.NewKeyFromArmored(string(pubkey))
	if err != nil {
		log.Fatalln(err)
		return false
	}
	signingKeyRing, err := crypto.NewKeyRing(publicKeyObj)
	if err != nil {
		log.Fatalln(err)
		return false
	}

	err = signingKeyRing.VerifyDetached(message, pgpSignature, crypto.GetUnixTime())
	if err != nil {
		log.Fatalln(err)
		return false
	}
	return true
}

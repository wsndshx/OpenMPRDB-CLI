package main

import (
	"errors"
	"io/ioutil"

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

// SignatureData 对指定文本进行签名, 返回签名后的内容
func SignatureData(text string) (string, error) {
	var privkey string
	err := db.QueryRow("SELECT private_key FROM Config").Scan(&privkey)
	if err != nil {
		return "", errors.New("无法读取本地私钥: " + err.Error())
	}
	armored, err := helper.SignCleartextMessageArmored(privkey, nil, text)
	if err != nil {
		return "", errors.New("签名时发生错误: " + err.Error())
	}
	return armored, nil
}

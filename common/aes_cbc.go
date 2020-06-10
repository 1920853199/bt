package common

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
)

type AesEncrypt struct{}

//
//func (a *AesEncrypt) validSecret(secret string) ([]byte, error) {
//	lenght := len(secret)
//	if lenght < 16 {
//		return nil, errors.New("长度有误")
//	}
//	return []byte(secret)[:16], nil
//}
//
//func (a *AesEncrypt) Encrypt(str []byte, secret string) (string, error) {
//	key, err := a.validSecret(secret)
//	if err != nil {
//		return "", err
//	}
//
//	block, err := aes.NewCipher(key)
//	if err != nil {
//		return "", err
//	}
//
//	blockSize := block.BlockSize()
//	origData := a.pKCS5Padding(str, blockSize)
//
//	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
//	crypted := make([]byte, len(origData))
//	blockMode.CryptBlocks(crypted, origData)
//
//	return base64.StdEncoding.EncodeToString(crypted), nil
//}

func (a *AesEncrypt) Decrypt(crypted, secret []byte) ([]byte, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	crypted, err = base64.StdEncoding.DecodeString(string(crypted))
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, secret[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = a.pKCS5UnPadding(origData)

	return origData, nil
}

func (a *AesEncrypt) pKCS5UnPadding(origData []byte) []byte {
	lenght := len(origData)
	unpadding := int(origData[lenght-1])
	return origData[:(lenght - unpadding)]
}

//func (a *AesEncrypt) pKCS5Padding(ciphertext []byte, blockSize int) []byte {
//	padding := blockSize - len(ciphertext)%blockSize
//	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
//	return append(ciphertext, padtext...)
//}

package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// getAESKey 从环境变量获取 AES 密钥，避免硬编码
func getAESKey() string {
	key := os.Getenv("AES_KEY")
	if key == "" {
		key = "this-is-32-byte-key-for-aes-256!" // 开发环境默认值
	}
	return key
}

// AESEncrypt 使用AES加密字符串
func AESEncrypt(plaintext string) (string, error) {
	key := []byte(getAESKey())
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", fmt.Errorf("invalid AES key size: %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 填充明文到块大小
	plaintextBytes := []byte(plaintext)
	plaintextBytes = pkcs7Pad(plaintextBytes, aes.BlockSize)

	// 创建加密块
	ciphertext := make([]byte, aes.BlockSize+len(plaintextBytes))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	// 加密
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintextBytes)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESDecrypt 使用AES解密字符串
func AESDecrypt(ciphertext string) (string, error) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Recovered from AES decryption panic: %v\n", r)
			err = fmt.Errorf("解密失败: %v", r)
		}
	}()

	key := []byte(getAESKey())
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", fmt.Errorf("invalid AES key size: %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	if len(ciphertextBytes) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := ciphertextBytes[:aes.BlockSize]
	ciphertextBytes = ciphertextBytes[aes.BlockSize:]

	// 解密
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertextBytes, ciphertextBytes)

	// 去除填充
	ciphertextBytes, err = pkcs7Unpad(ciphertextBytes)
	if err != nil {
		return "", err
	}

	return string(ciphertextBytes), err
}

// pkcs7Pad 填充数据到块大小
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7Unpad 去除填充数据
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("data is empty")
	}
	padding := int(data[len(data)-1])
	if padding > len(data) {
		return nil, errors.New("invalid padding")
	}
	return data[:len(data)-padding], nil
}

// EncryptionMd5 MD5加密 (仅用于旧密码兼容性检查，新密码应使用 HashPassword)
func EncryptionMd5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// HashPassword 使用 bcrypt 哈希密码 (H3: 替代 MD5)
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 验证密码是否匹配 bcrypt 哈希
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CheckPasswordCompat 兼容旧 MD5 和新 bcrypt 密码
// 先尝试 bcrypt，失败则回退到 MD5 对比
func CheckPasswordCompat(password, storedHash string) bool {
	// 尝试 bcrypt
	if CheckPassword(password, storedHash) {
		return true
	}
	// 回退到 MD5 兼容
	return storedHash == EncryptionMd5(password)
}

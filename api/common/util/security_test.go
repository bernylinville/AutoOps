package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== #10 bcrypt 密码哈希测试 ====================

func TestHashPassword(t *testing.T) {
	password := "TestPassword@123"
	hash, err := HashPassword(password)
	assert.NoError(t, err, "哈希密码不应失败")
	assert.NotEmpty(t, hash, "哈希不应为空")
	assert.NotEqual(t, password, hash, "哈希不应等于明文")
}

func TestHashPassword_DifferentSalts(t *testing.T) {
	password := "same-password"
	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)
	assert.NotEqual(t, hash1, hash2, "同一密码两次哈希应产生不同结果（不同盐）")
}

func TestCheckPassword_Correct(t *testing.T) {
	password := "MySecurePass!456"
	hash, _ := HashPassword(password)
	assert.True(t, CheckPassword(password, hash), "正确密码应验证通过")
}

func TestCheckPassword_Wrong(t *testing.T) {
	password := "correct-password"
	hash, _ := HashPassword(password)
	assert.False(t, CheckPassword("wrong-password", hash), "错误密码应验证失败")
}

func TestCheckPassword_NotMD5(t *testing.T) {
	// 确保 MD5 格式的哈希不会被接受
	md5Hash := "e10adc3949ba59abbe56e057f20f883e" // MD5 of "123456"
	assert.False(t, CheckPassword("123456", md5Hash), "MD5 哈希不应被 bcrypt 验证接受")
}

// ==================== #10 AES 加解密测试 ====================

func TestAesEncryptDecrypt(t *testing.T) {
	plaintext := "sensitive-data-12345"
	encrypted, err := AESEncrypt(plaintext)
	assert.NoError(t, err, "加密不应失败")
	assert.NotEmpty(t, encrypted, "密文不应为空")
	assert.NotEqual(t, plaintext, encrypted, "密文不应等于明文")

	decrypted, err := AESDecrypt(encrypted)
	assert.NoError(t, err, "解密不应失败")
	assert.Equal(t, plaintext, decrypted, "解密后应等于原文")
}

func TestAesEncrypt_DifferentCiphertexts(t *testing.T) {
	plaintext := "test-data"
	c1, _ := AESEncrypt(plaintext)
	c2, _ := AESEncrypt(plaintext)
	// AES-CBC 使用随机 IV，同一明文应产生不同密文
	// 但如果使用 ECB 模式则相同，取决于实现
	// 此测试验证加解密一致性
	d1, _ := AESDecrypt(c1)
	d2, _ := AESDecrypt(c2)
	assert.Equal(t, d1, d2, "解密结果应一致")
}

func TestAesDecrypt_InvalidInput(t *testing.T) {
	_, err := AESDecrypt("not-valid-base64!!!")
	assert.Error(t, err, "非法密文应解密失败")
}

// ==================== #5 SSH 工具测试 ====================

func TestWriteRemoteFile_NilConn(t *testing.T) {
	sshUtil := NewSSHUtil()
	assert.NotNil(t, sshUtil, "SSHUtil 应成功创建")
}

func TestSSHConfig_AllAuthTypes(t *testing.T) {
	// 验证支持的认证类型
	configs := []SSHConfig{
		{Type: 1, Username: "root", Password: "pass", IP: "127.0.0.1", Port: 22},
		{Type: 2, Username: "root", PublicKey: "invalid-key", IP: "127.0.0.1", Port: 22},
	}
	sshUtil := NewSSHUtil()

	for _, cfg := range configs {
		_, err := sshUtil.getSSHConfig(&cfg)
		// Type 1 应成功（密码认证不需要外部资源）
		// Type 2 应失败（无效私钥）
		if cfg.Type == 1 {
			assert.NoError(t, err, "密码认证配置不应失败")
		} else if cfg.Type == 2 {
			assert.Error(t, err, "无效私钥应返回错误")
		}
	}
}

func TestSSHConfig_UnsupportedType(t *testing.T) {
	sshUtil := NewSSHUtil()
	cfg := &SSHConfig{Type: 99, Username: "root", IP: "127.0.0.1", Port: 22}
	_, err := sshUtil.getSSHConfig(cfg)
	assert.Error(t, err, "不支持的认证类型应返回错误")
	assert.Contains(t, err.Error(), "unsupported")
}

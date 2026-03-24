package jwt

import (
	"os"
	"testing"

	"dodevops-api/api/system/model"

	"github.com/stretchr/testify/assert"
)

// ==================== #10 JWT Token 测试 ====================

func TestGenerateAndValidateToken(t *testing.T) {
	admin := model.SysAdmin{
		Username: "testuser",
		Nickname: "Test User",
		Email:    "test@example.com",
	}
	admin.ID = 1

	token, err := GenerateTokenByAdmin(admin)
	assert.NoError(t, err, "生成 token 不应失败")
	assert.NotEmpty(t, token, "token 不应为空")

	jwtAdmin, err := ValidateToken(token)
	assert.NoError(t, err, "验证合法 token 不应失败")
	assert.Equal(t, uint(1), jwtAdmin.ID)
	assert.Equal(t, "testuser", jwtAdmin.Username)
	assert.Equal(t, "Test User", jwtAdmin.Nickname)
}

func TestValidateToken_Empty(t *testing.T) {
	_, err := ValidateToken("")
	assert.Error(t, err, "空 token 应返回错误")
	assert.Contains(t, err.Error(), "absent")
}

func TestValidateToken_Invalid(t *testing.T) {
	_, err := ValidateToken("this-is-not-a-jwt-token")
	assert.Error(t, err, "无效 token 应返回错误")
}

func TestValidateToken_TamperedPayload(t *testing.T) {
	admin := model.SysAdmin{Username: "admin"}
	admin.ID = 1
	token, _ := GenerateTokenByAdmin(admin)

	// 篡改 token 的 payload 部分
	tampered := token[:10] + "TAMPERED" + token[18:]
	_, err := ValidateToken(tampered)
	assert.Error(t, err, "篡改后的 token 应被拒绝")
}

func TestJWTSecret_NotDefault(t *testing.T) {
	if os.Getenv("GIN_MODE") == "release" {
		t.Skip("跳过：生产模式下无默认密钥")
	}
	assert.NotEmpty(t, Secret, "JWT Secret 不应为空")
}

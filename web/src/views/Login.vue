<template>
    <div class="login-container">
        <div class="login-card">
            <div class="card-header">
                <h2 class="card-title">AutoOps</h2>
                <span class="card-subtitle">运维管理系统</span>
            </div>

            <!-- 表单 -->
            <el-form ref="loginFormRef" :rules="rules" :model="loginForm">
                <el-form-item prop="username">
                    <el-input
                        v-model="loginForm.username"
                        placeholder="请输入账号"
                        clearable
                        size="large"
                    />
                </el-form-item>

                <el-form-item prop="password">
                    <el-input
                        v-model="loginForm.password"
                        placeholder="请输入密码"
                        type="password"
                        show-password
                        clearable
                        size="large"
                    />
                </el-form-item>

                <el-form-item prop="image">
                    <div class="captcha-row">
                        <el-input
                            v-model="loginForm.image"
                            placeholder="请输入验证码"
                            maxlength="6"
                            clearable
                            size="large"
                        />
                        <div class="captcha-box" @click="getCaptcha">
                            <el-image :src="image" class="captcha-img" />
                        </div>
                    </div>
                </el-form-item>

                <el-form-item>
                    <el-button class="login-btn" type="primary" size="large" @click="loginBtn">登 录</el-button>
                    <el-button class="reset-btn" size="large" @click="resetLoginForm">重 置</el-button>
                </el-form-item>
            </el-form>
        </div>
    </div>
</template>

<script>
export default {
    name: "Login",
    data() {
        return {
            image: '',
            rules: {
                username: [{ required: true, message: "请输入账号", trigger: "blur" }],
                password: [{ required: true, message: "请输入密码", trigger: "blur" }],
                image:    [{ required: true, message: "请输入验证码", trigger: "blur" }]
            },
            loginForm: {
                username: '',
                password: '',
                image: '',
                idKey: ''
            }
        }
    },
    methods: {
        async getCaptcha() {
            const { data: res } = await this.$api.captcha()
            if (res.code !== 200) {
                this.$message.error(res.message)
            } else {
                this.image = res.data.image
                this.loginForm.idKey = res.data.idKey
            }
        },
        loginBtn() {
            this.$refs.loginFormRef.validate(async valid => {
                if (valid) {
                    const { data: res } = await this.$api.login(this.loginForm)
                    if (res.code !== 200) {
                        this.$message.error(res.message)
                    } else {
                        this.$message.success("登录成功")
                        this.$store.commit('saveSysAdmin', res.data.sysAdmin)
                        this.$store.commit('saveToken', res.data.token)
                        this.$store.commit('saveLeftMenuList', res.data.leftMenuList)
                        this.$store.commit('savePermissionList', res.data.permissionList)
                        await this.$router.push("/home")
                    }
                } else {
                    return false
                }
            })
        },
        resetLoginForm() {
            this.$refs.loginFormRef.resetFields()
        }
    },
    created() {
        this.getCaptcha()
    }
}
</script>

<style lang="less" scoped>
.login-container {
    width: 100%;
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: url('../assets/image/背景.jpg') center / cover no-repeat;
    font-family: var(--ao-font-family);
}

// 卡片 — 深色半透明，不用玻璃拟态
.login-card {
    position: relative;
    z-index: 1;
    width: 400px;
    padding: 40px 36px 32px;
    background: rgba(15, 23, 42, 0.88);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: var(--ao-radius-lg);
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
    text-align: center;
}

// 标题行
.card-header {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: 10px;
    margin-bottom: 32px;
}

.card-title {
    font-size: 22px;
    font-weight: 700;
    color: #f1f5f9;
    margin: 0;
    letter-spacing: 1px;
}

.card-subtitle {
    font-size: 16px;
    font-weight: 400;
    color: rgba(241, 245, 249, 0.7);
    letter-spacing: 1px;
}

// 输入框 — 使用 Element Plus 标准样式，仅微调背景
:deep(.el-input__wrapper) {
    background: rgba(255, 255, 255, 0.95);
    border-radius: var(--ao-radius);
}

:deep(.el-input__inner) {
    color: var(--ao-text-primary);
}

// 验证码行
.captcha-row {
    display: flex;
    gap: 10px;
    width: 100%;

    .el-input {
        flex: 1;
    }
}

.captcha-box {
    flex-shrink: 0;
    width: 110px;
    height: 40px;
    border-radius: var(--ao-radius);
    overflow: hidden;
    cursor: pointer;
    border: 1px solid rgba(255, 255, 255, 0.12);
    transition: border-color var(--ao-transition);

    &:hover {
        border-color: var(--ao-primary);
    }
}

.captcha-img {
    width: 100%;
    height: 100%;
    display: block;
}

// 表单间距
:deep(.el-form-item) {
    margin-bottom: 20px;

    &:last-child {
        margin-bottom: 0;
    }

    .el-form-item__error {
        font-size: 12px;
        color: var(--ao-danger);
        padding-top: 4px;
    }
}

// 登录按钮 — 使用 Element Plus primary 蓝色
.login-btn {
    width: calc(100% - 100px);
    font-size: 15px;
    font-weight: 600;
    letter-spacing: 2px;
}

// 重置按钮
.reset-btn {
    width: 86px;
    margin-left: 10px !important;
    border: 1px solid rgba(255, 255, 255, 0.15);
    background: rgba(255, 255, 255, 0.08);
    color: rgba(148, 163, 184, 0.9);
    font-size: 14px;
    letter-spacing: 1px;

    &:hover {
        border-color: rgba(255, 255, 255, 0.25);
        color: #e2e8f0;
        background: rgba(255, 255, 255, 0.12);
    }
}
</style>

package controller

import (
	"fmt"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// checkOriginAllowlist WebSocket Origin白名单校验
func checkOriginAllowlist(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	allowed := os.Getenv("ALLOWED_ORIGINS")
	if allowed == "" {
		allowed = "http://localhost:8080,http://localhost:3000"
	}
	for _, a := range strings.Split(allowed, ",") {
		if strings.TrimSpace(a) == origin {
			return true
		}
	}
	return false
}

type noEarlyResponseWriter struct {
	gin.ResponseWriter
	headerWritten bool
}

type hijackedResponseWriter struct {
	conn net.Conn
}

func (w *noEarlyResponseWriter) WriteHeader(code int) {
	if !w.headerWritten {
		w.headerWritten = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *noEarlyResponseWriter) WriteHeaderNow() {
	if !w.headerWritten {
		w.headerWritten = true
		w.ResponseWriter.WriteHeaderNow()
	}
}

func (w *hijackedResponseWriter) Header() http.Header {
	return http.Header{}
}

func (w *hijackedResponseWriter) Write(data []byte) (int, error) {
	return w.conn.Write(data)
}

func (w *hijackedResponseWriter) WriteHeader(statusCode int) {
	statusText := http.StatusText(statusCode)
	if statusText == "" {
		statusText = "status code " + strconv.Itoa(statusCode)
	}
	w.conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, statusText)))
}

type CmdbHostSSHController struct {
	hostSSHService service.CmdbHostSSHServiceInterface
}

func NewCmdbHostSSHController(hostSSHService service.CmdbHostSSHServiceInterface) *CmdbHostSSHController {
	return &CmdbHostSSHController{hostSSHService: hostSSHService}
}

// ConnectTerminal 连接SSH终端
func (c *CmdbHostSSHController) ConnectTerminal(ctx *gin.Context) {
	hostID := ctx.Param("id")
	log.Printf("开始处理WebSocket连接请求, hostID: %s", hostID)

	// 解析主机ID
	id, err := strconv.ParseUint(hostID, 10, 32)
	if err != nil {
		log.Printf("无效的主机ID: %s, 错误: %v", hostID, err)
		ctx.String(http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 使用标准WebSocket升级（严格Origin校验）
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: checkOriginAllowlist,
	}

	// 优先从URL参数获取token
	token := ctx.Query("token")
	if token == "" {
		// 如果URL中没有token，检查Authorization头
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("缺少token或Authorization头")
			ctx.String(http.StatusUnauthorized, "缺少token或Authorization头")
			return
		}

		// 验证Bearer令牌格式
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			log.Println("无效的Authorization头格式")
			ctx.String(http.StatusUnauthorized, "无效的Authorization头格式")
			return
		}
		token = authHeader[7:]
	}
	_, err = jwt.ValidateToken(token)
	if err != nil {
		log.Printf("令牌验证失败: %v", err)
		ctx.String(http.StatusUnauthorized, "令牌验证失败")
		return
	}

	// 升级WebSocket连接
	wsConn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		ctx.String(http.StatusInternalServerError, "WebSocket升级失败")
		return
	}

	// 连接SSH终端
	log.Println("尝试连接SSH终端")
	webSSH, err := c.hostSSHService.ConnectTerminal(ctx, uint(id))
	if err != nil {
		log.Printf("SSH连接失败: %v", err)
		wsConn.Close()
		ctx.String(http.StatusInternalServerError, "SSH连接失败")
		return
	}

	// 连接WebSocket
	if err := webSSH.Connect(wsConn); err != nil {
		log.Printf("WebSSH连接失败: %v", err)
		webSSH.Close()
		wsConn.Close()
		ctx.String(http.StatusInternalServerError, "WebSSH连接失败")
		return
	}

	// SSH连接测试通过
	log.Println("SSH连接测试通过")
	log.Printf("SSH连接测试通过, WebSocket和SSH连接已建立, 远程地址: %s", wsConn.RemoteAddr().String())

	// 设置defer确保资源释放
	defer func() {
		webSSH.Close()
		wsConn.Close()
		log.Printf("WebSocket连接已关闭, hostID: %d", id)
	}()

	// 等待客户端断开连接（替代 select{} 避免goroutine泄露）
	<-ctx.Request.Context().Done()
}

// ExecuteCommand 执行命令
// @Summary 执行SSH命令
// @Description 在SSH终端执行命令
// @Tags CMDB主机SSH
// @Accept  json
// @Produce  json
// @Param id path uint true "主机ID"
// @Param command query string true "命令"
// @Success 200 {object} result.Result
// @Router /api/v1/cmdb/hostssh/command/{id} [get]
// @Security ApiKeyAuth
func (c *CmdbHostSSHController) ExecuteCommand(ctx *gin.Context) {
	hostID := ctx.Param("id")
	command := ctx.Query("command")

	id, err := strconv.ParseUint(hostID, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的主机ID")
		return
	}

	output, err := c.hostSSHService.ExecuteCommand(ctx, uint(id), command)
	if err != nil {
		result.Failed(ctx, http.StatusInternalServerError, "执行命令失败: "+err.Error())
		return
	}

	result.Success(ctx, gin.H{
		"hostId":  hostID,
		"command": command,
		"output":  output,
	})
}

// UploadFile 上传文件到SSH服务器
// @Summary 上传文件到SSH服务器
// @Description 上传本地文件到远程SSH服务器
// @Tags CMDB主机SSH
// @Accept multipart/form-data
// @Produce json
// @Param id path uint true "主机ID"
// @Param file formData file true "要上传的文件"
// @Param destPath formData string true "远程服务器目标路径"
// @Success 200 {object} result.Result
// @Router /api/v1/cmdb/hostssh/upload/{id} [post]
// @Security ApiKeyAuth
func (c *CmdbHostSSHController) UploadFile(ctx *gin.Context) {
	hostID := ctx.Param("id")
	file, err := ctx.FormFile("file")
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "获取上传文件失败: "+err.Error())
		return
	}

	destPath := ctx.PostForm("destPath")
	if destPath == "" {
		result.Failed(ctx, http.StatusBadRequest, "目标路径不能为空")
		return
	}

	// 创建独立临时目录防止路径穿越和并发冲突
	tempDir, err := os.MkdirTemp("", "ssh_upload_*")
	if err != nil {
		log.Printf("创建临时目录失败: %v", err)
		result.Failed(ctx, http.StatusInternalServerError, "创建临时目录失败: "+err.Error())
		return
	}
	defer os.RemoveAll(tempDir) // 请求结束时清理整个临时目录
	log.Printf("创建临时目录: %s", tempDir)

	// 安全处理文件名：只取基础文件名，防止路径穿越
	safeFilename := filepath.Base(file.Filename)
	if safeFilename == "." || safeFilename == "/" || safeFilename == "" {
		safeFilename = "upload"
	}
	tempFilePath := filepath.Join(tempDir, safeFilename)
	log.Printf("尝试保存文件到: %s (原始文件名: %s)", tempFilePath, file.Filename)

	// 保存文件并设置权限
	log.Printf("保存文件: %s (大小: %d bytes)", file.Filename, file.Size)
	if err := ctx.SaveUploadedFile(file, tempFilePath); err != nil {
		log.Printf("保存文件失败: %v", err)
		result.Failed(ctx, http.StatusInternalServerError, "保存临时文件失败: "+err.Error())
		return
	}

	// 设置文件权限为0644
	if err := os.Chmod(tempFilePath, 0644); err != nil {
		log.Printf("设置文件权限失败: %v", err)
		result.Failed(ctx, http.StatusInternalServerError, "设置文件权限失败: "+err.Error())
		return
	}

	// 严格验证文件是否保存成功
	if fi, err := os.Stat(tempFilePath); err != nil {
		log.Printf("文件保存验证失败: %v", err)
		result.Failed(ctx, http.StatusInternalServerError, "文件保存验证失败: "+err.Error())
		return
	} else {
		log.Printf("文件保存成功: %s (大小: %d bytes, 权限: %v)", 
			tempFilePath, fi.Size(), fi.Mode())
	}

	// 强制同步文件系统
	if file, err := os.Open(tempFilePath); err == nil {
		file.Sync()
		file.Close()
		log.Printf("已同步文件系统: %s", tempFilePath)
	} else {
		log.Printf("无法打开文件进行同步: %v", err)
		result.Failed(ctx, http.StatusInternalServerError, "无法验证文件状态: "+err.Error())
		return
	}

	// 再次验证文件存在
	if _, err := os.Stat(tempFilePath); err != nil {
		log.Printf("文件在同步后丢失: %v", err)
		result.Failed(ctx, http.StatusInternalServerError, "文件同步后丢失: "+err.Error())
		return
	}

	// 转换为uint
	id, err := strconv.ParseUint(hostID, 10, 32)
	if err != nil {
		result.Failed(ctx, http.StatusBadRequest, "无效的主机ID")
		return
	}

	// 验证文件在调用服务前仍然存在
	if _, err := os.Stat(tempFilePath); err != nil {
		log.Printf("文件在调用服务前已不存在: %v", err)
		result.Failed(ctx, http.StatusInternalServerError, "临时文件已丢失: "+err.Error())
		return
	}

	// 调用服务上传文件
	log.Printf("开始上传文件到远程服务器: %s -> %s", tempFilePath, destPath)
	err = c.hostSSHService.UploadFile(ctx, uint(id), tempFilePath, destPath)
	if err != nil {
		log.Printf("文件上传失败: %v", err)
		// 保留临时文件用于调试
		log.Printf("保留临时文件用于调试: %s", tempFilePath)
		result.Failed(ctx, http.StatusInternalServerError, "文件上传失败: "+err.Error())
		return
	} else {
		log.Printf("文件上传成功: %s -> %s", tempFilePath, destPath)
	}

	// 临时文件由 defer os.RemoveAll(tempDir) 自动清理

	result.Success(ctx, gin.H{
		"hostId":    hostID,
		"fileName":  file.Filename,
		"destPath": destPath,
		"message":  "文件上传成功",
	})
}

// =============================================================================
// 模块: Terminal 终端 (agent/app/api/v2/terminal.go)
// 文件: terminal.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package v2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/service"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	"github.com/1Panel-dev/1Panel/agent/utils/ssh"
	"github.com/1Panel-dev/1Panel/agent/utils/terminal"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// @Tags Terminal
// @Summary Ws local terminal
// @Param command query string false "command"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// WsLocalTerminal  打开本机 WebSocket 终端（直接进服务器 shell）
// ============================================================
// 参数:
//   - c (*gin.Context) — HTTP 请求上下文
// 流程:
//   1. 复用通用 SSH 会话启动逻辑
//   2. 用 loadLocalConn 拿本地 SSH 连接
//   3. 进入本地 shell
// 调用: 前端 "本机终端" -> this; this -> runSSHSession
// ============================================================

// @Router /hosts/terminal/local [get]
func (b *BaseApi) WsLocalTerminal(c *gin.Context) {
	b.runSSHSession(c, loadLocalConn, c.DefaultQuery("command", ""))
}

// @Tags Terminal
// @Summary Ws host SSH
// @Param id query integer false "id"
// @Param command query string false "command"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// WsHostSSH  打开远程主机的 WebSocket 终端（通过 SSH）
// ============================================================
// 参数:
//   - c (*gin.Context) — query 中带 host id 和可选 command
// 流程:
//   1. 从 query 拿 host id，查出主机信息
//   2. 用 newHostSSHClient 建立 SSH 客户端
//   3. 复用 runSSHSession 走通用 SSH 终端流程
// 调用: 前端 "远程主机终端" -> this; this -> runSSHSession -> newHostSSHClient
// ============================================================

// @Router /hosts/terminal/ssh [get]
func (b *BaseApi) WsHostSSH(c *gin.Context) {
	b.runSSHSession(c, func() (*ssh.SSHClient, error) {
		hostID, _ := strconv.Atoi(c.DefaultQuery("id", "0"))
		if hostID <= 0 {
			return nil, errors.New("missing host id")
		}
		host, err := service.GetHostInfo(uint(hostID))
		return newHostSSHClient(host, err)
	}, c.DefaultQuery("command", ""))
}

// @Tags Terminal
// @Summary Ws container terminal
// @Param cols query integer false "cols"
// @Param rows query integer false "rows"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// WsContainerTerminal  打开容器内部 WebSocket 终端（docker exec）
// ============================================================
// 参数:
//   - c (*gin.Context) — query 中带 cols/rows/source
// 流程:
//   1. 准备 WebSocket + 终端尺寸
//   2. 根据 source (redis/ollama/container/database) 拼出 docker exec 命令
//   3. 启动伪终端，把容器内 shell 推到前端
//   4. 阻塞到任一端关闭
// 调用: 前端 "容器终端" -> this; this -> loadContainerTerminalCommand
// ============================================================

// @Router /hosts/terminal/container [get]
func (b *BaseApi) WsContainerTerminal(c *gin.Context) {
	wsConn, cols, rows, ok := prepareTerminalSession(c)
	if !ok {
		return
	}
	defer wsConn.Close()

	slave, err := loadContainerTerminalCommand(c)
	if wshandleError(wsConn, err) {
		return
	}
	defer slave.Close()

	tty, err := terminal.NewLocalWsSession(cols, rows, wsConn, slave, false)
	if wshandleError(wsConn, err) {
		return
	}

	quitChan := make(chan bool, 3)
	tty.Start(quitChan)
	go slave.Wait(quitChan)

	<-quitChan

	global.LOG.Info("websocket finished")
	closeTerminalConn(wsConn)
}

// ============================================================
// prepareTerminalSession  准备一个 WebSocket 终端会话（公共前置步骤）
// ============================================================
// 作用:
//   - 校验 WebSocket 握手
//   - 升级 HTTP 为 WebSocket
//   - 拒绝演示服务器
//   - 解析 cols/rows（终端列数/行数）
// 参数:
//   - c (*gin.Context) — HTTP 请求上下文
// 返回:
//   - (*websocket.Conn, cols, rows, ok) — 连接、终端列、终端行、是否成功
// 流程:
//   1. 校验 WebSocket Upgrade
//   2. 升级连接
//   3. 演示环境直接拒绝
//   4. 解析 query 里的 cols/rows
// 调用: WsLocalTerminal / WsHostSSH / WsContainerTerminal -> this
// ============================================================
func prepareTerminalSession(c *gin.Context) (*websocket.Conn, int, int, bool) {
	if !websocket.IsWebSocketUpgrade(c.Request) {
		helper.Success(c)
		return nil, 0, 0, false
	}
	wsConn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		global.LOG.Errorf("gin context http handler failed, err: %v", err)
		return nil, 0, 0, false
	}

	if global.CONF.Base.IsDemo {
		if wshandleError(wsConn, errors.New("   demo server, prohibit this operation!")) {
			return nil, 0, 0, false
		}
	}

	cols, err := strconv.Atoi(c.DefaultQuery("cols", "80"))
	if wshandleError(wsConn, errors.WithMessage(err, "invalid param cols in request")) {
		return nil, 0, 0, false
	}
	rows, err := strconv.Atoi(c.DefaultQuery("rows", "40"))
	if wshandleError(wsConn, errors.WithMessage(err, "invalid param rows in request")) {
		return nil, 0, 0, false
	}
	return wsConn, cols, rows, true
}

// ============================================================
// runSSHSession  通用的 SSH 终端会话启动器（WebSocket ↔ SSH 双向桥接）
// ============================================================
// 参数:
//   - c (*gin.Context) — HTTP 上下文
//   - connect (func() (*ssh.SSHClient, error)) — 拿到 SSH 客户端的回调（本地/远端不同实现）
//   - command (string) — 可选的初始命令
// 流程:
//   1. 准备 WebSocket
//   2. 调 connect 拿 SSH 客户端
//   3. 用 terminal.NewLogicSshWsSession 桥接 WebSocket 和 SSH
//   4. 阻塞到 quit
// 调用: WsLocalTerminal / WsHostSSH -> this
// ============================================================
func (b *BaseApi) runSSHSession(c *gin.Context, connect func() (*ssh.SSHClient, error), command string) {
	wsConn, cols, rows, ok := prepareTerminalSession(c)
	if !ok {
		return
	}
	defer wsConn.Close()

	client, clientErr := connect()
	if wshandleError(wsConn, errors.WithMessage(clientErr, "failed to set up the connection. Please check the host information")) {
		return
	}
	defer client.Close()

	sws, err := terminal.NewLogicSshWsSession(cols, rows, client.Client, wsConn, command)
	if wshandleError(wsConn, err) {
		return
	}
	defer sws.Close()

	quitChan := make(chan bool, 3)
	sws.Start(quitChan)
	go sws.Wait(quitChan)

	<-quitChan

	closeTerminalConn(wsConn)
}

// ============================================================
// closeTerminalConn  优雅地关闭一个 WebSocket 终端连接
// ============================================================
// 流程:
//   1. 给前端发一个 Close 控制帧
//   2. 1 秒超时（避免阻塞）
// 调用: WsLocalTerminal / WsHostSSH / WsContainerTerminal -> this
// ============================================================
func closeTerminalConn(wsConn *websocket.Conn) {
	dt := time.Now().Add(time.Second)
	_ = wsConn.WriteControl(websocket.CloseMessage, nil, dt)
}

// ============================================================
// newHostSSHClient  根据数据库里的 Host 记录建立 SSH 客户端
// ============================================================
// 参数:
//   - host (*model.Host) — 主机信息（地址/端口/账号/认证方式）
//   - err (error) — 上一步查 host 的错误
// 返回:
//   - (*ssh.SSHClient, error) — 包装好的 SSH 客户端
// 流程:
//   1. err 不为空就直接透传
//   2. 把 host 字段映射到 ssh.ConnInfo
//   3. 调 ssh.NewClient 真正建连
// 调用: WsHostSSH -> this
// ============================================================
func newHostSSHClient(host *model.Host, err error) (*ssh.SSHClient, error) {
	if err != nil {
		return nil, errors.WithMessage(err, "load host info by id failed")
	}
	connInfo := ssh.ConnInfo{
		Addr:       host.Addr,
		Port:       int(host.Port),
		User:       host.User,
		AuthMode:   host.AuthMode,
		Password:   host.Password,
		PrivateKey: []byte(host.PrivateKey),
	}
	if len(host.PassPhrase) != 0 {
		connInfo.PassPhrase = []byte(host.PassPhrase)
	}
	return ssh.NewClient(connInfo)
}

// ============================================================
// loadContainerTerminalCommand  根据 source 拼出"进入容器内 shell"的命令
// ============================================================
// 参数:
//   - c (*gin.Context) — query 中带 source
// 返回:
//   - (*terminal.LocalCommand, error) — 用 docker exec 包装出的本地命令
// 流程:
//   1. 按 source 分发到不同 loader（redis/ollama/container/database）
//   2. 拿到 init 命令后用 terminal.NewCommand 包成 docker exec
// 调用: WsContainerTerminal -> this
// ============================================================
func loadContainerTerminalCommand(c *gin.Context) (*terminal.LocalCommand, error) {
	source := c.Query("source")
	var (
		initCmd []string
		err     error
	)
	switch source {
	case "redis", "redis-cluster":
		initCmd, err = loadRedisInitCmd(c, source)
	case "ollama":
		initCmd, err = loadOllamaInitCmd(c)
	case "container":
		initCmd, err = loadContainerInitCmd(c)
	case "database":
		initCmd, err = loadDatabaseInitCmd(c)
	default:
		return nil, fmt.Errorf("not support such source %s", source)
	}
	if err != nil {
		return nil, err
	}
	return terminal.NewCommand("docker", initCmd...)
}

// ============================================================
// loadRedisInitCmd  拼出"进入 redis-cli"的 docker exec 命令
// ============================================================
// 参数:
//   - c (*gin.Context) — query 中带 name/from
//   - redisType (string) — "redis" 或 "redis-cluster"
// 流程:
//   1. 查数据库拿到地址/端口/密码
//   2. from=local 时直接 exec 容器 + redis-cli
//   3. 否则用 1Panel-redis-cli-tools 工具容器连上去
// 调用: loadContainerTerminalCommand -> this
// ============================================================
func loadRedisInitCmd(c *gin.Context, redisType string) ([]string, error) {
	name := c.Query("name")
	from := c.Query("from")
	commands := []string{"exec", "-it"}
	database, err := databaseService.Get(name)
	if err != nil {
		return nil, fmt.Errorf("no such database in db, err: %v", err)
	}
	if from == "local" {
		redisInfo, err := appInstallService.LoadConnInfo(dto.OperationWithNameAndType{Name: name, Type: redisType})
		if err != nil {
			return nil, fmt.Errorf("no such app in db, err: %v", err)
		}
		name = redisInfo.ContainerName
		commands = append(commands, []string{name, "redis-cli"}...)
		if len(database.Password) != 0 {
			commands = append(commands, []string{"-a", database.Password, "--no-auth-warning"}...)
		}
	} else {
		name = "1Panel-redis-cli-tools"
		commands = append(commands, []string{name, "redis-cli", "-h", database.Address, "-p", fmt.Sprintf("%v", database.Port)}...)
		if len(database.Password) != 0 {
			commands = append(commands, []string{"-a", database.Password, "--no-auth-warning"}...)
		}
	}
	return commands, nil
}

// ============================================================
// loadOllamaInitCmd  拼出"ollama run <model>" 的 docker exec 命令
// ============================================================
// 参数:
//   - c (*gin.Context) — query 中带 name（模型名）
// 流程:
//   1. 校验模型名（防注入）
//   2. 查出 ollama 容器名
//   3. 拼出 exec -it <容器> ollama run <模型>
// 调用: loadContainerTerminalCommand -> this
// ============================================================
func loadOllamaInitCmd(c *gin.Context) ([]string, error) {
	name := c.Query("name")
	if cmd.CheckIllegal(name) {
		return nil, fmt.Errorf("ollama model %s contains illegal characters", name)
	}
	ollamaInfo, err := appInstallService.LoadConnInfo(dto.OperationWithNameAndType{Name: "", Type: "ollama"})
	if err != nil {
		return nil, fmt.Errorf("no such app in db, err: %v", err)
	}
	containerName := ollamaInfo.ContainerName
	return []string{"exec", "-it", containerName, "ollama", "run", name}, nil
}

// ============================================================
// loadContainerInitCmd  拼出"docker exec -it <container> <command>"
// ============================================================
// 参数:
//   - c (*gin.Context) — query 中带 containerid/command/user
// 流程:
//   1. 校验所有入参（防命令注入）
//   2. 缺参数报错
//   3. 可选地追加 -u <user>
// 调用: loadContainerTerminalCommand -> this
// ============================================================
func loadContainerInitCmd(c *gin.Context) ([]string, error) {
	containerID := c.Query("containerid")
	command := c.Query("command")
	user := c.Query("user")
	if cmd.CheckIllegal(user, containerID, command) {
		return nil, fmt.Errorf("the command contains illegal characters. command: %s, user: %s, containerID: %s", command, user, containerID)
	}
	if len(command) == 0 || len(containerID) == 0 {
		return nil, fmt.Errorf("error param of command: %s or containerID: %s", command, containerID)
	}
	commands := []string{"exec", "-it", containerID, command}
	if len(user) != 0 {
		commands = []string{"exec", "-it", "-u", user, containerID, command}
	}

	return commands, nil
}

// ============================================================
// loadDatabaseInitCmd  拼出"进入数据库 CLI"的 docker exec 命令
// ============================================================
// 参数:
//   - c (*gin.Context) — query 中带 database/databaseType
// 流程:
//   1. 查应用安装表拿到数据库连接信息
//   2. 按 mysql / mariadb / mongodb / postgresql 拼不同命令
//   3. 带上账号密码
// 调用: loadContainerTerminalCommand -> this
// ============================================================
func loadDatabaseInitCmd(c *gin.Context) ([]string, error) {
	database := c.Query("database")
	databaseType := c.Query("databaseType")
	if len(databaseType) == 0 {
		return nil, fmt.Errorf("error param of database: %s or database type: %s", database, databaseType)
	}
	databaseConn, err := appInstallService.LoadConnInfo(dto.OperationWithNameAndType{Type: databaseType, Name: database})
	if err != nil {
		return nil, fmt.Errorf("no such database in db, err: %v", err)
	}
	if len(databaseConn.ContainerName) == 0 {
		return nil, fmt.Errorf("no such database container for database: %s or database type: %s", database, databaseType)
	}
	commands := []string{"exec", "-it", databaseConn.ContainerName}
	switch databaseType {
	case "mysql", "mysql-cluster":
		commands = append(commands, []string{"mysql", "-uroot", "-p" + databaseConn.Password}...)
	case "mariadb":
		commands = append(commands, []string{"mariadb", "-uroot", "-p" + databaseConn.Password}...)
	case "mongodb":
		commands = append(commands, []string{
			"mongosh",
			"--username", databaseConn.Username,
			"--password", databaseConn.Password,
			"--authenticationDatabase", "admin",
		}...)
	case "postgresql", "postgresql-cluster":
		commands = []string{"exec", "-e", fmt.Sprintf("PGPASSWORD=%s", databaseConn.Password), "-it", databaseConn.ContainerName, "psql", "-t", "-U", databaseConn.Username}
	}

	return commands, nil
}

// ============================================================
// wshandleError  WebSocket 错误处理工具：把错误推到前端并返回 true
// ============================================================
// 参数:
//   - ws (*websocket.Conn) — 当前 WebSocket
//   - err (error) — 待处理错误
// 返回:
//   - bool — true 表示有错误（处理过），false 表示无错
// 流程:
//   1. 尝试发 Close 控制帧
//   2. 失败时改发一条 WsMsg JSON 文本
// 调用: WsContainerTerminal / runSSHSession -> this
// ============================================================
func wshandleError(ws *websocket.Conn, err error) bool {
	if err != nil {
		global.LOG.Errorf("handler ws faled:, err: %v", err)
		dt := time.Now().Add(time.Second)
		if ctlerr := ws.WriteControl(websocket.CloseMessage, []byte(err.Error()), dt); ctlerr != nil {
			wsData, err := json.Marshal(terminal.WsMsg{
				Type: terminal.WsMsgCmd,
				Data: base64.StdEncoding.EncodeToString([]byte(err.Error())),
			})
			if err != nil {
				_ = ws.WriteMessage(websocket.TextMessage, []byte("{\"type\":\"cmd\",\"data\":\"failed to encoding to json\"}"))
			} else {
				_ = ws.WriteMessage(websocket.TextMessage, wsData)
			}
		}
		return true
	}
	return false
}

// ============================================================
// upGrader  WebSocket 升级器（HTTP -> WS 握手用），允许任意来源
// ============================================================
// 字段:
//   - ReadBufferSize (int) — 读缓冲 4KB
//   - WriteBufferSize (int) — 写缓冲 16KB
//   - CheckOrigin (func) — 跨域检查（这里是全放行）
// ============================================================
var upGrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 16384,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

package cmd

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/zh-five/xdaemon"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"google.golang.org/grpc"

	"goflylivechat/agent"
	"goflylivechat/common"
	"goflylivechat/middleware"
	"goflylivechat/router"
	"goflylivechat/tools"
	"goflylivechat/ws"
)

var (
	port     string
	grpcPort string
	daemon   bool
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "Start HTTP and gRPC services",
	Example: "gochat server -p 8082 --grpc-port 9090",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func init() {
	serverCmd.PersistentFlags().StringVarP(&port, "port", "p", "8081", "HTTP port to listen on")
	serverCmd.PersistentFlags().StringVar(&grpcPort, "grpc-port", "9090", "gRPC port to listen on")
	serverCmd.PersistentFlags().BoolVarP(&daemon, "daemon", "d", false, "Run as daemon process")
}

// run 输入命令参数，输出为 HTTP 与 gRPC 服务启动结果，目的在于以统一入口初始化配置、依赖、观测与优雅停机流程。
func run() {
	appConfig := common.GetAppConfig()
	appConfig.HTTPPort = port
	appConfig.GRPCPort = grpcPort

	configureDaemonMode()

	shutdownTracing := initializeTracing(appConfig)
	defer func() {
		if shutdownTracing != nil {
			_ = shutdownTracing(context.Background())
		}
	}()

	initializeCache()
	tools.InitMetrics()
	agent.ConfigureDefaultRegistry(appConfig.AgentHeartbeatTTL)

	grpcServer, grpcListener := mustStartGRPCServer(appConfig)
	defer grpcServer.GracefulStop()

	agentClient, clientError := agent.NewClient("127.0.0.1:"+appConfig.GRPCPort, appConfig)
	if clientError != nil {
		log.Printf("初始化 agent gRPC 客户端失败: %v", clientError)
	} else {
		agent.SetDefaultClient(agentClient)
		defer func() {
			_ = agentClient.Close()
		}()

		agentDispatcher, dispatcherError := agent.NewDispatcher(agentClient, appConfig, agent.DispatcherHooks{
			OnAssigned: ws.NotifyVisitorAgentAssigned,
		})
		if dispatcherError != nil {
			log.Fatalf("初始化 agent 调度器失败: %v", dispatcherError)
		}
		agent.SetDefaultDispatcher(agentDispatcher)
		defer func() {
			_ = agentDispatcher.Close()
		}()
	}

	engine := buildHTTPRouter(appConfig)
	httpServer := &http.Server{
		Addr:         "0.0.0.0:" + appConfig.HTTPPort,
		Handler:      engine,
		ReadTimeout:  appConfig.HTTPReadTimeout,
		WriteTimeout: appConfig.HTTPWriteTimeout,
		IdleTimeout:  appConfig.HTTPIdleTimeout,
	}

	go func() {
		log.Printf("HTTP service listening on http://0.0.0.0:%s", appConfig.HTTPPort)
		if listenError := httpServer.ListenAndServe(); listenError != nil && !errors.Is(listenError, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务启动失败: %v", listenError)
		}
	}()

	log.Printf("gRPC service listening on %s", grpcListener.Addr().String())
	waitForShutdown(httpServer, grpcServer, appConfig.ShutdownTimeout)
}

// configureDaemonMode 输入为空，输出为守护进程配置结果，目的在于兼容现有 daemon 启动方式。
func configureDaemonMode() {
	if !daemon {
		return
	}
	logFilePath := ""
	if dir, err := os.Getwd(); err == nil {
		logFilePath = dir + "/logs/"
	}
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		if mkdirError := os.MkdirAll(logFilePath, 0777); mkdirError != nil {
			log.Println(mkdirError.Error())
		}
	}
	daemonRuntime := xdaemon.NewDaemon(logFilePath + "gofly.log")
	daemonRuntime.MaxCount = 10
	daemonRuntime.Run()
}

// initializeTracing 输入应用配置，输出为 tracing 关闭函数，目的在于按配置启用链路追踪并在退出时释放资源。
func initializeTracing(appConfig common.AppConfig) func(context.Context) error {
	if !appConfig.EnableTracing {
		return nil
	}
	shutdownTracing, tracingError := tools.InitJaeger(appConfig.JaegerEndpoint)
	if tracingError != nil {
		log.Printf("初始化 tracing 失败: %v", tracingError)
		return nil
	}
	return shutdownTracing
}

// initializeCache 输入为空，输出为多级缓存初始化结果，目的在于让 Redis 不可用时自动退化到内存缓存。
func initializeCache() {
	rawRedisConfig := common.GetRedisConf()
	redisConfig := tools.RedisConfig{
		Host:     rawRedisConfig.Host,
		Port:     rawRedisConfig.Port,
		Password: rawRedisConfig.Password,
		DB:       rawRedisConfig.DB,
		PoolSize: rawRedisConfig.PoolSize,
	}
	if cacheError := tools.InitMultiLevelCache(redisConfig); cacheError != nil {
		log.Printf("初始化多级缓存失败: %v", cacheError)
	}
}

// buildHTTPRouter 输入应用配置，输出为 Gin 引擎实例，目的在于集中装配中间件、静态资源和业务路由。
func buildHTTPRouter(appConfig common.AppConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestID())
	engine.Use(tools.MetricsMiddleware())
	engine.Use(middleware.NewMidLogger())
	engine.Use(middleware.CrossSite)
	engine.Use(middleware.Timeout(appConfig.RequestTimeout))
	engine.Use(middleware.NewRateLimiter(appConfig.RateLimitPerSecond, appConfig.RateLimitBurst).Middleware())
	engine.Use(middleware.SessionHandler())
	if appConfig.EnableTracing {
		engine.Use(otelgin.Middleware("goflylivechat"))
	}

	engine.StaticFile("/favicon.ico", "./static/images/logo.png")
	engine.Static("/static", "./static")

	router.InitViewRouter(engine)
	router.InitApiRouter(engine)

	tools.NewLimitQueue()
	ws.CleanVisitorExpire()
	go ws.WsServerBackend()

	return engine
}

// waitForShutdown 输入 HTTP 服务、gRPC 服务与关闭超时，输出为优雅停机结果，目的在于统一处理信号退出和资源回收。
func waitForShutdown(httpServer *http.Server, grpcServer *grpc.Server, shutdownTimeout time.Duration) {
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignals

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if shutdownError := httpServer.Shutdown(shutdownContext); shutdownError != nil {
		log.Printf("HTTP 服务关闭异常: %v", shutdownError)
	}
	grpcServer.GracefulStop()
}

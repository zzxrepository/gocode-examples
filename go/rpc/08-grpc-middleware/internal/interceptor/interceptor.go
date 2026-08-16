package interceptor

import (
	"context"
	// log 用在 Recovery 中打印 panic 堆栈。
	"log"
	// os/path/filepath 用来创建日志目录和拼接日志文件路径。
	"os"
	"path/filepath"
	// runtime/debug 可以拿到当前 goroutine 的堆栈。
	"runtime/debug"
	// strings 用来去掉 authorization 里的 "bearer " 前缀。
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Auth 是认证拦截器。
//
// 一元拦截器的签名固定为：
// func(ctx, req, info, handler) (resp, err)
//
// handler 表示后续拦截器或真正的业务方法。
// 如果这里直接返回错误，业务方法就不会执行。
func Auth(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	// 从 ctx 中取出客户端 metadata。
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	// 客户端传来的 authorization 形如：bearer grpc.auth.token。
	// 服务端只校验 token 部分。
	token := strings.TrimPrefix(first(md.Get("authorization")), "bearer ")
	if token != "grpc.auth.token" {
		return nil, status.Error(codes.Unauthenticated, "invalid bearer token")
	}
	// 认证通过，继续执行后续逻辑。
	return handler(ctx, req)
}

// Recovery 是 panic 恢复拦截器。
// 它放在拦截器链最前面，目的是兜住后续拦截器或业务方法里的 panic。
func Recovery(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			// 打印 panic 和堆栈，方便排查问题。
			log.Printf("panic recovered: %v\n%s", r, debug.Stack())
			// 把 panic 转成 gRPC 错误返回给客户端，避免服务进程崩溃。
			err = status.Errorf(codes.Internal, "panic recovered: %v", r)
		}
	}()
	return handler(ctx, req)
}

// Logging 返回一个日志拦截器。
// 它记录 RPC 方法名、最终状态码和耗时。
func Logging(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// 记录开始时间，handler 执行后计算耗时。
		start := time.Now()
		resp, err := handler(ctx, req)
		logger.Info("finished unary call",
			zap.String("method", info.FullMethod),
			zap.String("code", status.Code(err).String()),
			zap.Duration("duration", time.Since(start)),
		)
		return resp, err
	}
}

// NewLogger 创建 zap logger。
// 这里把日志写到 log/debug.log，并使用 lumberjack 做简单的日志切割。
func NewLogger() (*zap.Logger, error) {
	// 确保 log 目录存在。
	if err := os.MkdirAll("log", 0o755); err != nil {
		return nil, err
	}

	// 使用生产环境 JSON 日志格式，并把时间编码成人类可读的 ISO8601。
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder

	// zapcore 负责决定日志“写到哪里”和“怎么编码”。
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config),
		zapcore.AddSync(&lumberjack.Logger{
			Filename:  filepath.Join("log", "debug.log"),
			MaxSize:   10,
			LocalTime: true,
		}),
		zap.InfoLevel,
	)
	return zap.New(core, zap.AddCaller()), nil
}

// first 获取 metadata 某个 key 的第一个值。
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

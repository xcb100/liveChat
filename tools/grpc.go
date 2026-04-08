package tools

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const grpcRequestIDMetadataKey = "x-request-id"

// UnaryServerLoggingInterceptor 输入为空，输出为 gRPC 服务端一元拦截器，目的在于记录方法、耗时、状态码与请求标识。
func UnaryServerLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startedAt := time.Now()
		requestID := getRequestIDFromIncomingContext(ctx)
		clientAddress := ""
		if peerInfo, ok := peer.FromContext(ctx); ok && peerInfo.Addr != nil {
			clientAddress = peerInfo.Addr.String()
		}

		response, handlerError := handler(ctx, request)
		log.Printf(
			"grpc_server method=%s request_id=%s peer=%s code=%s duration=%s",
			info.FullMethod,
			requestID,
			clientAddress,
			status.Code(handlerError).String(),
			time.Since(startedAt).String(),
		)
		return response, handlerError
	}
}

// UnaryClientLoggingInterceptor 输入为空，输出为 gRPC 客户端一元拦截器，目的在于传播 request id 并记录调用耗时与状态码。
func UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request interface{}, reply interface{}, connection *grpc.ClientConn, invoker grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
		startedAt := time.Now()
		requestID := getRequestIDFromContext(ctx)
		if requestID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, grpcRequestIDMetadataKey, requestID)
		}

		invokeError := invoker(ctx, method, request, reply, connection, callOptions...)
		log.Printf(
			"grpc_client method=%s request_id=%s code=%s duration=%s",
			method,
			requestID,
			status.Code(invokeError).String(),
			time.Since(startedAt).String(),
		)
		return invokeError
	}
}

// getRequestIDFromContext 输入上下文，输出为请求标识，目的在于从 HTTP 透传上下文中提取 request id。
func getRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestIDValue := ctx.Value("request_id")
	requestID, _ := requestIDValue.(string)
	return requestID
}

// getRequestIDFromIncomingContext 输入 gRPC 上下文，输出为请求标识，目的在于从元数据或本地上下文中提取 request id。
func getRequestIDFromIncomingContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if metadataValue, ok := metadata.FromIncomingContext(ctx); ok {
		requestIDs := metadataValue.Get(grpcRequestIDMetadataKey)
		if len(requestIDs) > 0 && requestIDs[0] != "" {
			return requestIDs[0]
		}
	}
	return getRequestIDFromContext(ctx)
}

// CodeToString 输入 gRPC 错误，输出为状态码字符串，目的在于为测试和日志提供稳定状态文案。
func CodeToString(err error) string {
	if err == nil {
		return codes.OK.String()
	}
	return status.Code(err).String()
}

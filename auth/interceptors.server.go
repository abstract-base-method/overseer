package auth

import (
	"context"
	"overseer/common"
	"slices"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryServerAuthFunc(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if slices.Contains(reflectionMethods, info.FullMethod) {
		common.GetLogger("server.auth.urnary").Info("auth bypassed for reflection method", "method", info.FullMethod)
		return handler(ctx, req)
	}

	md := metadata.ExtractIncoming(ctx)
	peer, ok := peer.FromContext(ctx)
	if !ok {
		common.GetLogger("server.auth.urnary").Error("failed to retrieve peer from context", "method", info.FullMethod)
		return nil, status.Error(codes.Unauthenticated, "failed to retrieve peer information")
	}
	if peer == nil {
		common.GetLogger("server.auth.urnary").Error("peer is nil", "method", info.FullMethod)
		return nil, status.Error(codes.Unauthenticated, "peer was nil")
	}

	userInfo, err := authenticateFromMetadata(md)
	if err != nil {
		common.GetLogger("server.auth.urnary").Error("failed to authenticate", "method", info.FullMethod, "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to authenticate")
	}

	authCtx, err := common.SetContextInformation(ctx, userInfo)
	if err != nil {
		common.GetLogger("server.auth.urnary").Error("failed to set context information", "method", info.FullMethod, "error", err)
		return nil, status.Error(codes.Unauthenticated, "failed to establish context information")
	}

	common.GetLogger("server.auth.urnary").Debug("authenticated call", userInfo.LoggingContext("method", info.FullMethod)...)
	return handler(authCtx, req)
}

func StreamServerAuthFunc(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if slices.Contains(reflectionMethods, info.FullMethod) {
		common.GetLogger("server.auth.stream").Info("auth bypassed for reflection method", "method", info.FullMethod)
		return handler(srv, stream)
	}

	md := metadata.ExtractIncoming(stream.Context())
	peer, ok := peer.FromContext(stream.Context())
	if !ok {
		common.GetLogger("server.auth.stream").Error("failed to retrieve peer from context", "method", info.FullMethod)
		return status.Error(codes.Unauthenticated, "failed to retrieve peer information")
	}
	if peer == nil {
		common.GetLogger("server.auth.stream").Error("peer is nil", "method", info.FullMethod)
		return status.Error(codes.Unauthenticated, "peer was nil")
	}

	userInfo, err := authenticateFromMetadata(md)
	if err != nil {
		common.GetLogger("server.auth.stream").Error("failed to authenticate", "method", info.FullMethod, "error", err)
		return status.Error(codes.Unauthenticated, "failed to authenticate")
	}

	authCtx, err := common.SetContextInformation(stream.Context(), userInfo)
	if err != nil {
		common.GetLogger("server.auth.stream").Error("failed to set context information", "method", info.FullMethod, "error", err)
		return status.Error(codes.Unauthenticated, "failed to establish context information")
	}

	wrapped := middleware.WrapServerStream(stream)
	wrapped.WrappedContext = authCtx

	common.GetLogger("server.auth.stream").Debug("authenticated stream", userInfo.LoggingContext("method", info.FullMethod)...)
	return handler(srv, wrapped)
}

func authenticateFromMetadata(md metadata.MD) (*common.OverseerContextInformation, error) {
	if common.GetConfiguration().Server.EnableSystemToken {
		if systemToken, ok := md[systemTokenKey]; ok {
			if systemToken[0] == common.GetConfiguration().Server.SystemToken {
				return systemContextInformation, nil
			}
		}
	}
	// todo: implement authentication
	return nil, status.Error(codes.Unauthenticated, "authentication not implemented")
}

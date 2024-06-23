package common

import (
	"context"
	v1 "overseer/build/go"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type overseerContextKey string

type OverseerContextInformation struct {
	User  *v1.User
	Actor *v1.Actor
}

func (c *OverseerContextInformation) LoggingContext(additionalKV ...interface{}) []interface{} {
	userContext := make([]interface{}, 0)
	if c.User != nil {
		userContext = append(userContext, "user", c.User.GetUid())
	}
	if c.Actor != nil {
		userContext = append(userContext, "actor", c.Actor.GetUid())
		userContext = append(userContext, "source", c.Actor.GetSource().String())
		userContext = append(userContext, "source_identity", c.Actor.GetSourceIdentity())
	}
	return append(userContext, additionalKV...)
}

const (
	actiorInformationKey overseerContextKey = "user:info"
)

func GetContextInformation(ctx context.Context) (*OverseerContextInformation, error) {
	info, ok := ctx.Value(actiorInformationKey).(*OverseerContextInformation)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "context information not found")
	}
	if info == nil {
		return nil, status.Error(codes.Unauthenticated, "context information was nil")
	}
	return info, nil
}

func SetContextInformation(ctx context.Context, info *OverseerContextInformation) (context.Context, error) {
	if info == nil {
		return nil, status.Error(codes.InvalidArgument, "context information is nil")
	}
	if info.User == nil {
		return nil, status.Error(codes.InvalidArgument, "context information user is nil")
	}
	return context.WithValue(ctx, actiorInformationKey, info), nil
}

package grpcx

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/apperr"
)

// NewErrorInterceptor converts handler errors into gRPC status errors:
//   - *AppError      -> status with its GRPCCode + safe message
//   - existing status -> passed through unchanged
//   - anything else  -> Internal (details not leaked)
//
// Install it as the OUTERMOST interceptor so it also passes through the
// InvalidArgument produced by the validation interceptor.
func NewErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		if ae, ok := apperr.FromError(err); ok {
			return nil, status.Error(ae.GRPCCode(), ae.Message())
		}
		if _, isStatus := status.FromError(err); isStatus {
			return nil, err
		}
		return nil, status.Error(codes.Internal, "internal server error")
	}
}

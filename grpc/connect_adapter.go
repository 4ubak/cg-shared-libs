package grpc

import (
	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConvertConnectToGRPCError converts Connect error codes to gRPC status codes.
// This function provides a unified way to convert Connect errors (used in HTTP/Connect handlers)
// to standard gRPC status errors (used in gRPC servers).
//
// The mapping follows the standard Connect to gRPC error code conversion:
//   - connect.CodeNotFound -> codes.NotFound
//   - connect.CodeInvalidArgument -> codes.InvalidArgument
//   - connect.CodeAlreadyExists -> codes.AlreadyExists
//   - connect.CodePermissionDenied -> codes.PermissionDenied
//   - connect.CodeUnauthenticated -> codes.Unauthenticated
//   - connect.CodeResourceExhausted -> codes.ResourceExhausted
//   - connect.CodeFailedPrecondition -> codes.FailedPrecondition
//   - connect.CodeAborted -> codes.Aborted
//   - connect.CodeOutOfRange -> codes.OutOfRange
//   - connect.CodeUnimplemented -> codes.Unimplemented
//   - connect.CodeInternal -> codes.Internal
//   - connect.CodeUnavailable -> codes.Unavailable
//   - connect.CodeDataLoss -> codes.DataLoss
//   - All other errors -> codes.Unknown
//
// If the error is not a Connect error, it will be converted to codes.Unknown
// with the original error message.
func ConvertConnectToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		return status.Error(codes.Unknown, err.Error())
	}

	var grpcCode codes.Code
	switch connectErr.Code() {
	case connect.CodeNotFound:
		grpcCode = codes.NotFound
	case connect.CodeInvalidArgument:
		grpcCode = codes.InvalidArgument
	case connect.CodeAlreadyExists:
		grpcCode = codes.AlreadyExists
	case connect.CodePermissionDenied:
		grpcCode = codes.PermissionDenied
	case connect.CodeUnauthenticated:
		grpcCode = codes.Unauthenticated
	case connect.CodeResourceExhausted:
		grpcCode = codes.ResourceExhausted
	case connect.CodeFailedPrecondition:
		grpcCode = codes.FailedPrecondition
	case connect.CodeAborted:
		grpcCode = codes.Aborted
	case connect.CodeOutOfRange:
		grpcCode = codes.OutOfRange
	case connect.CodeUnimplemented:
		grpcCode = codes.Unimplemented
	case connect.CodeInternal:
		grpcCode = codes.Internal
	case connect.CodeUnavailable:
		grpcCode = codes.Unavailable
	case connect.CodeDataLoss:
		grpcCode = codes.DataLoss
	default:
		grpcCode = codes.Unknown
	}

	return status.Error(grpcCode, connectErr.Message())
}

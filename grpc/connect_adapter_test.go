package grpc

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConvertConnectToGRPCError(t *testing.T) {
	tests := []struct {
		name        string
		input       error
		expectedCode codes.Code
		expectError bool
	}{
		{
			name:        "nil error",
			input:       nil,
			expectedCode: codes.OK,
			expectError: false,
		},
		{
			name:        "Connect NotFound error",
			input:       connect.NewError(connect.CodeNotFound, errors.New("not found")),
			expectedCode: codes.NotFound,
			expectError: true,
		},
		{
			name:        "Connect InvalidArgument error",
			input:       connect.NewError(connect.CodeInvalidArgument, errors.New("invalid argument")),
			expectedCode: codes.InvalidArgument,
			expectError: true,
		},
		{
			name:        "Connect AlreadyExists error",
			input:       connect.NewError(connect.CodeAlreadyExists, errors.New("already exists")),
			expectedCode: codes.AlreadyExists,
			expectError: true,
		},
		{
			name:        "Connect PermissionDenied error",
			input:       connect.NewError(connect.CodePermissionDenied, errors.New("permission denied")),
			expectedCode: codes.PermissionDenied,
			expectError: true,
		},
		{
			name:        "Connect Unauthenticated error",
			input:       connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated")),
			expectedCode: codes.Unauthenticated,
			expectError: true,
		},
		{
			name:        "Connect ResourceExhausted error",
			input:       connect.NewError(connect.CodeResourceExhausted, errors.New("resource exhausted")),
			expectedCode: codes.ResourceExhausted,
			expectError: true,
		},
		{
			name:        "Connect FailedPrecondition error",
			input:       connect.NewError(connect.CodeFailedPrecondition, errors.New("failed precondition")),
			expectedCode: codes.FailedPrecondition,
			expectError: true,
		},
		{
			name:        "Connect Aborted error",
			input:       connect.NewError(connect.CodeAborted, errors.New("aborted")),
			expectedCode: codes.Aborted,
			expectError: true,
		},
		{
			name:        "Connect OutOfRange error",
			input:       connect.NewError(connect.CodeOutOfRange, errors.New("out of range")),
			expectedCode: codes.OutOfRange,
			expectError: true,
		},
		{
			name:        "Connect Unimplemented error",
			input:       connect.NewError(connect.CodeUnimplemented, errors.New("unimplemented")),
			expectedCode: codes.Unimplemented,
			expectError: true,
		},
		{
			name:        "Connect Internal error",
			input:       connect.NewError(connect.CodeInternal, errors.New("internal error")),
			expectedCode: codes.Internal,
			expectError: true,
		},
		{
			name:        "Connect Unavailable error",
			input:       connect.NewError(connect.CodeUnavailable, errors.New("unavailable")),
			expectedCode: codes.Unavailable,
			expectError: true,
		},
		{
			name:        "Connect DataLoss error",
			input:       connect.NewError(connect.CodeDataLoss, errors.New("data loss")),
			expectedCode: codes.DataLoss,
			expectError: true,
		},
		{
			name:        "Non-Connect error",
			input:       errors.New("some error"),
			expectedCode: codes.Unknown,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertConnectToGRPCError(tt.input)

			if tt.expectError {
				if result == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(result)
				if !ok {
					t.Fatal("result is not a gRPC status error")
				}
				if st.Code() != tt.expectedCode {
					t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
				}
			} else {
				if result != nil {
					t.Errorf("expected nil, got error: %v", result)
				}
			}
		})
	}
}

func TestConvertConnectToGRPCError_PreservesMessage(t *testing.T) {
	msg := "custom error message"
	connectErr := connect.NewError(connect.CodeNotFound, errors.New(msg))
	grpcErr := ConvertConnectToGRPCError(connectErr)

	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatal("result is not a gRPC status error")
	}

	if st.Message() != msg {
		t.Errorf("expected message %q, got %q", msg, st.Message())
	}
}

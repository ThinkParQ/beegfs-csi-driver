/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Modifications Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"fmt"
	"io"
	"k8s.io/klog/v2/klogr"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/go-logr/logr"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ctxRequestID = "reqID"

var requestIDCounter uint32

// Traditionally gRPC service handlers should return an error created by the
// package "google.golang.org/grpc/status".  However our gRPC service handlers
// should (but are not required to) return a "grpcError".  This allows logGRPC
// to log stack traces (provided by errors created by the package
// "github.com/pkg/errors") in a consistent way for all service handlers.
type grpcError struct {
	statusErr error // error of type created by google.golang.org/grpc/status
	cause     error // error of type created by github.com/pkg/errors
}

func newGrpcErrorFromCause(code codes.Code, cause error) grpcError {
	if cause == nil {
		cause = errors.New("")
	}
	statusErr := status.Error(code, cause.Error())
	return grpcError{statusErr: statusErr, cause: cause}
}
func newGrpcError(code codes.Code, msg string) grpcError {
	return newGrpcErrorFromCause(code, errors.New(msg))
}
func newGrpcErrorf(code codes.Code, format string, a ...interface{}) grpcError {
	return newGrpcError(code, fmt.Sprintf(format, a...))
}
func (e grpcError) Error() string {
	return e.statusErr.Error() + ": " + e.cause.Error()
}
func (e grpcError) Cause() error  { return e.cause }
func (e grpcError) Unwrap() error { return e.cause }
func (e grpcError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", e.Cause())
			io.WriteString(s, e.statusErr.Error())
			return
		}
		fallthrough
	case 's', 'q':
		io.WriteString(s, e.statusErr.Error())
	}
}
func (e grpcError) GetStatusErr() error { return e.statusErr }

func NewNonBlockingGRPCServer() *nonBlockingGRPCServer {
	return &nonBlockingGRPCServer{}
}

// NonBlocking server
type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {

	s.wg.Add(1)

	go s.serve(endpoint, ids, cs, ns)

	return
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {

	proto, addr, err := parseEndpoint(endpoint)
	if err != nil {
		LogFatal(nil, err, "Error parsing endpoint")
	}

	if proto == "unix" {
		addr = "/" + addr
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) { //nolint: vetshadow
			LogFatal(nil, err, "Failed to remove address", "address", addr)
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		LogFatal(nil, err, "Failed to listen")
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	s.server = server

	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}

	Logger(nil).Info("Listening for connections", "address", listener.Addr())

	if err = server.Serve(listener); err != nil {
		if err == grpc.ErrServerStopped {
			Logger(nil).Info(err.Error())
		} else {
			LogFatal(nil, err, "Fatal error")
		}
	}
}

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", errors.Errorf("invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	reqCtx := generateRequestContext(ctx)
	logLevel := LogLevelDebug
	// These GRPC methods are called very frequently. Filter them out so they only appear at higher log levels.
	if info.FullMethod == "/csi.v1.Identity/Probe" || info.FullMethod == "/csi.v1.Node/NodeGetCapabilities" {
		logLevel = LogLevelVerbose
	}

	Logger(reqCtx).V(logLevel).Info("GRPC call", "method", info.FullMethod, "request", protosanitizer.StripSecrets(req).String())
	resp, err := handler(reqCtx, req)
	if err != nil {
		LogError(reqCtx, err, "GRPC error", "method", info.FullMethod, "request", protosanitizer.StripSecrets(req).String())
		var grpcErr grpcError
		if errors.As(err, &grpcErr) {
			// only forward statusErr
			err = grpcErr.GetStatusErr()
		}
	} else {
		Logger(reqCtx).V(logLevel).Info("GRPC response", "response", protosanitizer.StripSecrets(resp).String(), "method", info.FullMethod)
	}
	return resp, err
}

// Generates a new context with a unique hexadecimal requestID to easily keep track of related logs
func generateRequestContext(parent context.Context) context.Context {
	return context.WithValue(parent, ctxRequestID, fmt.Sprintf("%04x", atomic.AddUint32(&requestIDCounter, 1)%0x10000))
}

// logger returns a klogr logger with as much context as possible
func Logger(ctx context.Context) logr.Logger {
	newLogger := klogr.New()
	if ctx != nil {
		if ctxRqId, ok := ctx.Value(ctxRequestID).(string); ok {
			newLogger = newLogger.WithValues(ctxRequestID, ctxRqId)
		}
	} else {
		newLogger = newLogger.WithValues("goroutine", "main")
	}
	return newLogger
}

func LogDebug(ctx context.Context, msg string, keysAndValues ...interface{}) {
	Logger(ctx).V(LogLevelDebug).Info(msg, keysAndValues...)
}

func LogVerbose(ctx context.Context, msg string, keysAndValues ...interface{}) {
	Logger(ctx).V(LogLevelVerbose).Info(msg, keysAndValues...)
}

func LogError(ctx context.Context, err error, msg string, keysAndValues ...interface{}) {
	Logger(ctx).WithValues("fullError", fmt.Sprintf("%+v", err)).Error(err, msg, keysAndValues...)
}

func LogFatal(ctx context.Context, err error, msg string, keysAndValues ...interface{}) {
	LogError(ctx, err, "Fatal: "+msg, keysAndValues...)
	os.Exit(255)
}

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package southbound

import (
	"context"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
	"io"
)

// Client P4runtime client interface
type Client interface {
	io.Closer
	Write(ctx context.Context, in *p4api.WriteRequest, opts ...grpc.CallOption) (*p4api.WriteResponse, error)
	// Read one or more P4 entities from the target.
	Read(ctx context.Context, in *p4api.ReadRequest, opts ...grpc.CallOption) (p4api.P4Runtime_ReadClient, error)

	// SetForwardingPipelineConfig Sets the P4 forwarding-pipeline config.
	SetForwardingPipelineConfig(ctx context.Context, in *p4api.SetForwardingPipelineConfigRequest, opts ...grpc.CallOption) (*p4api.SetForwardingPipelineConfigResponse, error)

	// GetForwardingPipelineConfig Gets the current P4 forwarding-pipeline config
	GetForwardingPipelineConfig(ctx context.Context, in *p4api.GetForwardingPipelineConfigRequest, opts ...grpc.CallOption) (*p4api.GetForwardingPipelineConfigResponse, error)
	StreamChannel(ctx context.Context, opts ...grpc.CallOption) (p4api.P4Runtime_StreamChannelClient, error)
	Capabilities(ctx context.Context, request *p4api.CapabilitiesRequest, opts ...grpc.CallOption) (*p4api.CapabilitiesResponse, error)
}

type client struct {
	grpcClient      *grpc.ClientConn
	p4runtimeClient p4api.P4RuntimeClient
}

func (c *client) StreamChannel(ctx context.Context, opts ...grpc.CallOption) (p4api.P4Runtime_StreamChannelClient, error) {
	log.Debugw("Getting stream channel")
	streamClient, err := c.p4runtimeClient.StreamChannel(ctx, opts...)
	return streamClient, errors.FromGRPC(err)
}

func (c *client) Read(ctx context.Context, request *p4api.ReadRequest, opts ...grpc.CallOption) (p4api.P4Runtime_ReadClient, error) {
	log.Debugw("Received Read request", "request", request)
	readClient, err := c.p4runtimeClient.Read(ctx, request, opts...)
	return readClient, errors.FromGRPC(err)
}

// Write Updates one or more P4 entities on the target.
func (c *client) Write(ctx context.Context, request *p4api.WriteRequest, opts ...grpc.CallOption) (*p4api.WriteResponse, error) {
	log.Debugw("Received Write request", "request", request)
	writeResponse, err := c.p4runtimeClient.Write(ctx, request, opts...)
	return writeResponse, errors.FromGRPC(err)
}

// SetForwardingPipelineConfig  sets the P4 forwarding-pipeline config.
func (c *client) SetForwardingPipelineConfig(ctx context.Context, request *p4api.SetForwardingPipelineConfigRequest, opts ...grpc.CallOption) (*p4api.SetForwardingPipelineConfigResponse, error) {
	log.Debugw("Received SetForwardingPipelineConfig request", "request", request)
	setForwardingPipelineConfigResponse, err := c.p4runtimeClient.SetForwardingPipelineConfig(ctx, request, opts...)
	return setForwardingPipelineConfigResponse, errors.FromGRPC(err)
}

// GetForwardingPipelineConfig  gets the current P4 forwarding-pipeline config.
func (c *client) GetForwardingPipelineConfig(ctx context.Context, request *p4api.GetForwardingPipelineConfigRequest, opts ...grpc.CallOption) (*p4api.GetForwardingPipelineConfigResponse, error) {
	log.Debugw("Received GetForwardingPipelineConfig request", "request", request)
	getForwardingPipelineConfigResponse, err := c.p4runtimeClient.GetForwardingPipelineConfig(ctx, request, opts...)
	return getForwardingPipelineConfigResponse, errors.FromGRPC(err)
}

// Capabilities discovers the capabilities of the P4Runtime server implementation.
func (c *client) Capabilities(ctx context.Context, request *p4api.CapabilitiesRequest, opts ...grpc.CallOption) (*p4api.CapabilitiesResponse, error) {
	log.Debugw("Received Capabilities request", "request", request)
	capabilitiesResponse, err := c.p4runtimeClient.Capabilities(ctx, request, opts...)
	return capabilitiesResponse, errors.FromGRPC(err)
}

func (c *client) Close() error {
	return c.grpcClient.Close()
}

var _ Client = &client{}

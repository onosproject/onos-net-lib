// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package provisioner client
package provisioner

import (
	"context"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	"google.golang.org/grpc/codes"
	"io"

	provisionerapi "github.com/onosproject/onos-api/go/onos/provisioner"
	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"google.golang.org/grpc"
)

var log = logging.GetLogger()

// Provisioner is a provisioner client interface
type Provisioner interface {
	Add(ctx context.Context, config *provisionerapi.Config) (*provisionerapi.AddConfigResponse, error)
	Delete(ctx context.Context, configID provisionerapi.ConfigID) (*provisionerapi.DeleteConfigResponse, error)
	Get(ctx context.Context, configID provisionerapi.ConfigID, includeArtifacts bool) (*provisionerapi.GetConfigResponse, error)
	Watch(ctx context.Context, ch chan *provisionerapi.Config, kind string, includeArtifacts bool) error
}

// NewProvisioner creates a new provisioner client
func NewProvisioner(provisionerEndpoint string, opts ...grpc.DialOption) (Provisioner, error) {
	if len(opts) == 0 {
		return nil, errors.New(errors.Invalid, "no opts given when creating provisioner client")
	}
	opts = append(opts,
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))
	conn, err := grpc.DialContext(context.Background(), provisionerEndpoint, opts...)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	client := provisionerapi.NewProvisionerServiceClient(conn)
	return &provisionerClient{
		client: client,
	}, nil
}

type provisionerClient struct {
	client provisionerapi.ProvisionerServiceClient
}

// Add adds config records and artifacts
func (p *provisionerClient) Add(ctx context.Context, config *provisionerapi.Config) (*provisionerapi.AddConfigResponse, error) {
	response, err := p.client.Add(ctx, &provisionerapi.AddConfigRequest{
		Config: config,
	})
	if err != nil {
		return nil, errors.FromGRPC(err)
	}
	return response, nil
}

// Delete deletes config records and artifacts
func (p *provisionerClient) Delete(ctx context.Context, id provisionerapi.ConfigID) (*provisionerapi.DeleteConfigResponse, error) {
	response, err := p.client.Delete(ctx, &provisionerapi.DeleteConfigRequest{
		ConfigID: id,
	})
	if err != nil {
		return nil, errors.FromGRPC(err)
	}
	return response, nil
}

// Get gets config records and artifacts based on a given config ID
func (p *provisionerClient) Get(ctx context.Context, id provisionerapi.ConfigID, includeArtifacts bool) (*provisionerapi.GetConfigResponse, error) {
	response, err := p.client.Get(ctx, &provisionerapi.GetConfigRequest{
		ConfigID:         id,
		IncludeArtifacts: includeArtifacts,
	})
	if err != nil {
		return nil, errors.FromGRPC(err)
	}
	return response, nil
}

// Watch watch config record and artifact changes
func (p *provisionerClient) Watch(ctx context.Context, ch chan *provisionerapi.Config, kind string, includeArtifacts bool) error {
	stream, err := p.client.List(ctx, &provisionerapi.ListConfigsRequest{
		Kind:             kind,
		IncludeArtifacts: includeArtifacts,
		Watch:            true,
	})
	if err != nil {
		return err
	}
	go func() {
		defer close(ch)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Warn(err)
				break
			}
			ch <- resp.Config
		}
	}()
	return nil
}

var _ Provisioner = &provisionerClient{}

// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package stratum contains utilities to interact with Stratum devices via gNMI and P4Runtime
package stratum

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var log = logging.GetLogger("stratum")

// GNMI is a connection context to a Stratum gNMI endpoint derived from onos.topo.GNMIEndpoint aspect
type GNMI struct {
	ID         topo.ID
	gnmiServer *topo.GNMIServer
	conn       *grpc.ClientConn
	Client     gnmi.GNMIClient
	Context    context.Context
}

// NewGNMI creates a new stratum gNMI connection context from the specified topo entity
// using its onos.topo.GNMIServer aspect
func NewGNMI(object *topo.Object, connectImmediately bool) (*GNMI, error) {
	d := &GNMI{
		ID:         object.ID,
		gnmiServer: &topo.GNMIServer{},
	}
	if err := object.GetAspect(d.gnmiServer); err != nil {
		return nil, err
	}
	if connectImmediately {
		return d, d.Connect()
	}
	return d, nil
}

// Connect establishes connection to the gNMI server
func (d *GNMI) Connect() error {
	log.Infof("%s: connecting to gNMI server...", d.ID)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: Deal with secrets
	}

	var err error
	endpoint := d.gnmiServer.Endpoint
	d.conn, err = grpc.Dial(fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port), opts...)
	if err != nil {
		return err
	}

	d.Client = gnmi.NewGNMIClient(d.conn)
	d.Context = context.Background()

	log.Infof("%s: connected to gNMI server", d.ID)
	return nil
}

// Disconnect terminates the gNMI connection
func (d *GNMI) Disconnect() error {
	return d.conn.Close()
}

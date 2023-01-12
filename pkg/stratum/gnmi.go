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
	ID       string
	endpoint *topo.Endpoint
	conn     *grpc.ClientConn
	Client   gnmi.GNMIClient
	Context  context.Context
}

// NewGNMI creates a new stratum gNMI connection context from the specified topo entity
// using its onos.topo.StratumAgents aspect GNMIEndpoint field
func NewGNMI(id string, endpoint *topo.Endpoint, connectImmediately bool) (*GNMI, error) {
	d := &GNMI{ID: id, endpoint: endpoint}
	if connectImmediately {
		return d, d.Connect()
	}
	return d, nil
}

// NewStratumGNMI creates a new stratum gNMI connection context from the specified topo entity
// using its onos.topo.StratumAgents aspect GNMIEndpoint field
func NewStratumGNMI(object *topo.Object, connectImmediately bool) (*GNMI, error) {
	stratumAgents := &topo.StratumAgents{}
	if err := object.GetAspect(stratumAgents); err != nil {
		return nil, err
	}
	return NewGNMI(string(object.ID), stratumAgents.GNMIEndpoint, connectImmediately)
}

// Connect establishes connection to the gNMI server
func (d *GNMI) Connect() error {
	log.Infof("%s: connecting to gNMI server...", d.ID)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO: Deal with secrets
	}

	var err error
	d.conn, err = grpc.Dial(fmt.Sprintf("%s:%d", d.endpoint.Address, d.endpoint.Port), opts...)
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

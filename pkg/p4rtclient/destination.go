// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package p4rtclient

import (
	"crypto/tls"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"time"
)

// Destination contains data used to connect to a server
type Destination struct {
	// Endpoint P4runtime server endpoint address
	Endpoint *topoapi.Endpoint
	// TargetID is the topology target entity ID
	TargetID topoapi.ID
	// Timeout is the connection timeout
	Timeout time.Duration
	// TLS config to use when connecting to target.
	TLS *topoapi.TLSOptions
	// DeviceID is the numerical ID to be used for p4runtime API interactions
	DeviceID uint64
	// RoleName a name given to a connection role
	RoleName string
}

func setCertificate(pathCert string, pathKey string) (tls.Certificate, error) {
	certificate, err := tls.LoadX509KeyPair(pathCert, pathKey)
	if err != nil {
		return tls.Certificate{}, errors.NewNotFound("could not load client key pair ", err)
	}
	return certificate, nil
}

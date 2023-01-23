// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package p4rtclient

import (
	"github.com/google/uuid"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/uri"
)

// ConnID connection ID
type ConnID string

// Conn connection interface
type Conn interface {
	Client
	ID() ConnID
	TargetID() topoapi.ID
	DeviceID() uint64
	RoleName() string
}

type conn struct {
	*client
	id       ConnID
	targetID topoapi.ID // topology entity ID
	roleName string
	deviceID uint64
}

// ID returns connection ID
func (c conn) ID() ConnID {
	return c.id
}

// TargetID returns P4 programmable target ID
func (c conn) TargetID() topoapi.ID {
	return c.targetID
}

// RoleName returns the connection role name
func (c conn) RoleName() string {
	return c.roleName
}

// DeviceID returns device ID
func (c conn) DeviceID() uint64 {
	return c.deviceID
}

func newConnID() ConnID {
	connID := ConnID(uri.NewURI(
		uri.WithScheme("uuid"),
		uri.WithOpaque(uuid.New().String())).String())
	return connID
}

func newConn(targetID topoapi.ID, p4rtClient *client, deviceID uint64, roleName string) Conn {
	conn := &conn{
		client:   p4rtClient,
		id:       newConnID(),
		targetID: targetID,
		deviceID: deviceID,
		roleName: roleName,
	}
	return conn
}

var _ Conn = conn{}

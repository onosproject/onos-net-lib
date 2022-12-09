// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package p4utils contains various utilities for working with P4Info and P4RT entities
package p4utils

import (
	gogo "github.com/gogo/protobuf/types"
	"github.com/onosproject/onos-api/go/onos/stratum"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/anypb"
	"io/ioutil"
	"time"
)

// TimeBasedElectionID returns election ID generated from the UnixNano timestamp
// High contains seconds, Low contains remaining nanos
func TimeBasedElectionID() *p4api.Uint128 {
	now := time.Now()
	t := now.UnixNano()
	return &p4api.Uint128{High: uint64(t / 1e9), Low: uint64(t % 1e9)}
}

// NewStratumRole produces a P4 Role with stratum.P4RoleConfig, and configured using the supplied parameters.
func NewStratumRole(roleName string, roleAgentIDMetaDataID uint32, roleAgentID []byte,
	receivesPacketIns bool, canPushPipeline bool) *p4api.Role {
	roleConfig := &stratum.P4RoleConfig{
		PacketInFilter: &stratum.P4RoleConfig_PacketFilter{
			MetadataId: roleAgentIDMetaDataID,
			Value:      roleAgentID,
		},
		ReceivesPacketIns: receivesPacketIns,
		CanPushPipeline:   canPushPipeline,
	}
	any, _ := gogo.MarshalAny(roleConfig)
	return &p4api.Role{
		Name: roleName,
		Config: &anypb.Any{
			TypeUrl: any.TypeUrl,
			Value:   any.Value,
		},
	}
}

// CreateMastershipArbitration returns stream message request with the specified election ID components
func CreateMastershipArbitration(electionID *p4api.Uint128, role *p4api.Role) *p4api.StreamMessageRequest {
	return &p4api.StreamMessageRequest{
		Update: &p4api.StreamMessageRequest_Arbitration{
			Arbitration: &p4api.MasterArbitrationUpdate{
				ElectionId: electionID,
				Role:       role,
			}}}
}

// LoadP4Info loads the specified file containing protoJSON representation of a P4Info and returns its descriptor
func LoadP4Info(path string) (*p4info.P4Info, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	info := &p4info.P4Info{}
	err = prototext.Unmarshal(data, info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// P4InfoBytes serializes the given P4 info structure into prototext bytes
func P4InfoBytes(info *p4info.P4Info) []byte {
	bytes, _ := prototext.Marshal(info)
	return bytes
}

// FindTable returns the named table from the specified P4Info; nil if not found
func FindTable(info *p4info.P4Info, tableName string) *p4info.Table {
	for _, table := range info.Tables {
		if table.Preamble.Name == tableName {
			return table
		}
	}
	return nil
}

// FindAction returns the named action from the specified P4Info; nil if not found
func FindAction(info *p4info.P4Info, actionName string) *p4info.Action {
	for _, action := range info.Actions {
		if action.Preamble.Name == actionName {
			return action
		}
	}
	return nil
}

// FindActionParam returns the named action from the specified P4Info; nil if not found
func FindActionParam(action *p4info.Action, paramName string) *p4info.Action_Param {
	for _, param := range action.Params {
		if param.Name == paramName {
			return param
		}
	}
	return nil
}

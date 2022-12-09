// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package p4utils contains various utilities for working with P4Info and P4RT entities
package p4utils

import (
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/protobuf/encoding/prototext"
	"io/ioutil"
	"time"
)

// CreateMastershipArbitration returns stream message request with the specified election ID components
func CreateMastershipArbitration(electionID *p4api.Uint128, role *p4api.Role) *p4api.StreamMessageRequest {
	return &p4api.StreamMessageRequest{
		Update: &p4api.StreamMessageRequest_Arbitration{
			Arbitration: &p4api.MasterArbitrationUpdate{
				ElectionId: electionID,
				Role:       role,
			}}}
}

// TimeBasedElectionID returns election ID generated from the UnixNano timestamp
// High contains seconds, Low contains remaining nanos
func TimeBasedElectionID() *p4api.Uint128 {
	now := time.Now()
	t := now.UnixNano()
	return &p4api.Uint128{High: uint64(t / 1e9), Low: uint64(t % 1e9)}
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

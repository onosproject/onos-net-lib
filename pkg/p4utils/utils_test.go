// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package p4utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadP4Info(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/p4info.txt")
	assert.NoError(t, err)
	assert.Equal(t, "tna", info.PkgInfo.Arch)

	assert.Len(t, info.Tables, 20)
	assert.Len(t, info.Actions, 40)
	assert.Len(t, info.ActionProfiles, 1)
	assert.Len(t, info.Meters, 1)
	assert.Len(t, info.Counters, 0)
	assert.Len(t, info.DirectMeters, 0)
	assert.Len(t, info.DirectCounters, 14)
	assert.Len(t, info.Digests, 0)
	assert.Len(t, info.Externs, 1)
	assert.Len(t, info.Registers, 0)
	assert.Len(t, info.ValueSets, 0)

	buf := P4InfoBytes(info)
	assert.True(t, len(buf) > 1024)

	// Test non-existent P4Info
	_, err = LoadP4Info("foobar.txt")
	assert.Error(t, err)

	// Test non-sensical P4Info
	_, err = LoadP4Info("utils_test.go")
	assert.Error(t, err)
}

func TestArbitration(t *testing.T) {
	eid := TimeBasedElectionID()
	mar := CreateMastershipArbitration(eid, NewStratumRole("foo", 1, []byte("\x03"), true, true))
	assert.NotNil(t, mar.GetArbitration())
	assert.Equal(t, eid.High, mar.GetArbitration().ElectionId.High)
	assert.Equal(t, eid.Low, mar.GetArbitration().ElectionId.Low)
}

func TestFindStuff(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/p4info.txt")
	assert.NoError(t, err)

	table := FindTable(info, "FabricIngress.acl.acl")
	assert.NotNil(t, table)

	field := FindTableMatchField(table, "eth_type")
	assert.NotNil(t, field)

	action := FindAction(info, "FabricIngress.acl.punt_to_cpu")
	assert.NotNil(t, action)

	param := FindActionParam(action, "set_role_agent_id")
	assert.NotNil(t, param)
}

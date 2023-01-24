// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package p4rtclient

import (
	"context"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-net-lib/pkg/p4utils"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"io"
	"testing"
)

const (
	targetPort = 9559
	targetHost = "localhost"
	targetID1  = "packet-switch-1"
	targetID2  = "packet-switch-2"
	deviceID1  = 1
	deviceID2  = 2
)

func newTestServer() *testServer {
	return &testServer{}
}

type testServer struct {
	northbound.Service
	p4api.UnimplementedP4RuntimeServer
}

func (s testServer) Write(ctx context.Context, request *p4api.WriteRequest) (*p4api.WriteResponse, error) {
	log.Infow("Write request is received", "request", request)
	response := &p4api.WriteResponse{}

	return response, nil
}

func (s testServer) Read(request *p4api.ReadRequest, server p4api.P4Runtime_ReadServer) error {
	log.Infow("Read request is received", "request", request)
	var entities []*p4api.Entity
	entity1 := &p4api.Entity{
		Entity: &p4api.Entity_TableEntry{
			TableEntry: &p4api.TableEntry{
				TableId: uint32(123),
			},
		},
	}
	entity2 := &p4api.Entity{
		Entity: &p4api.Entity_TableEntry{
			TableEntry: &p4api.TableEntry{
				TableId: uint32(124),
			},
		},
	}
	entities = append(entities, entity1)
	entities = append(entities, entity2)

	err := server.Send(&p4api.ReadResponse{
		Entities: entities,
	})
	if err != nil {
		log.Warnw("Cannot send read response", "error", err)
		return err
	}
	return nil
}

func (s testServer) SetForwardingPipelineConfig(ctx context.Context, request *p4api.SetForwardingPipelineConfigRequest) (*p4api.SetForwardingPipelineConfigResponse, error) {
	log.Infow("Set forwarding pipeline config request is received", "request", request)
	response := &p4api.SetForwardingPipelineConfigResponse{}
	return response, nil
}

func (s testServer) GetForwardingPipelineConfig(ctx context.Context, request *p4api.GetForwardingPipelineConfigRequest) (*p4api.GetForwardingPipelineConfigResponse, error) {
	log.Infow("Get forwarding pipeline config request is received", "request", request)
	response := &p4api.GetForwardingPipelineConfigResponse{}
	return response, nil
}

func (s testServer) sendPacketIn(payload []byte, server p4api.P4Runtime_StreamChannelServer) error {

	response := &p4api.StreamMessageResponse{
		Update: &p4api.StreamMessageResponse_Packet{
			Packet: &p4api.PacketIn{
				Payload: payload,
			},
		},
	}
	log.Info("Sending packet In", response)
	err := server.Send(response)
	if err != nil {
		log.Warn(err)
	}

	return nil

}

func (s testServer) StreamChannel(server p4api.P4Runtime_StreamChannelServer) error {
	ctx := server.Context()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// receive data from stream
		req, err := server.Recv()
		log.Info("Request server side:", req)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch v := req.Update.(type) {
		case *p4api.StreamMessageRequest_Arbitration:
			resp := p4api.StreamMessageResponse{
				Update: &p4api.StreamMessageResponse_Arbitration{
					Arbitration: &p4api.MasterArbitrationUpdate{
						ElectionId: &p4api.Uint128{
							Low:  v.Arbitration.ElectionId.Low,
							High: v.Arbitration.ElectionId.High,
						},
						Status: &status.Status{
							Code: 0,
						},
						DeviceId: v.Arbitration.DeviceId,
						Role:     req.GetArbitration().Role,
					},
				},
			}
			log.Info("Sending response")
			if err := server.Send(&resp); err != nil {
				log.Warn(err)
			}
		case *p4api.StreamMessageRequest_Packet:
			packetOutPayload := v.Packet.Payload
			err = s.sendPacketIn(packetOutPayload, server)
			if err != nil {
				log.Warn(err)
			}

		}

	}

}

func (s testServer) Capabilities(ctx context.Context, request *p4api.CapabilitiesRequest) (*p4api.CapabilitiesResponse, error) {
	log.Infow("Received capabilities request", "request", request)
	response := &p4api.CapabilitiesResponse{
		P4RuntimeApiVersion: "1.0.0",
	}
	return response, nil
}

// Register registers the Service with the gRPC server.
func (s testServer) Register(r *grpc.Server) {
	testServer := &testServer{}
	p4api.RegisterP4RuntimeServer(r, testServer)

}

func setup(t *testing.T, serverCfg *northbound.ServerConfig) *northbound.Server {
	s := northbound.NewServer(serverCfg)
	s.AddService(newTestServer())
	doneCh := make(chan error)

	go func() {
		err := s.Serve(func(started string) {
			t.Log("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	<-doneCh
	return s
}

func getTLSServerConfig(t *testing.T) *northbound.ServerConfig {
	return northbound.NewServerCfg(
		"",
		"",
		"",
		int16(targetPort),
		false,
		northbound.SecurityConfig{})
}

func getNonTLSServerConfig(t *testing.T) *northbound.ServerConfig {
	return northbound.NewServerCfg(
		"",
		"",
		"",
		int16(targetPort),
		true,
		northbound.SecurityConfig{})
}

func createTestDestination(t *testing.T, targetID string, deviceID uint64, insecure bool) *Destination {
	tlsOptions := &topoapi.TLSOptions{}
	if insecure {
		tlsOptions.Insecure = insecure
	}
	dest := &Destination{
		TargetID: topoapi.ID(targetID),
		DeviceID: uint64(1),
		TLS:      tlsOptions,
		Endpoint: &topoapi.Endpoint{
			Address: targetHost,
			Port:    targetPort,
		},
	}

	return dest
}

func TestConnManager_Get(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	connManager := NewConnManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestDestination(t, targetID1, deviceID1, true)
	target2 := createTestDestination(t, targetID2, deviceID2, true)

	_, err := connManager.Connect(ctx, target1)
	assert.NoError(t, err)
	conn1, err := connManager.GetByTarget(ctx, targetID1)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	_, err = connManager.Connect(ctx, target2)
	assert.NoError(t, err)
	conn2, err := connManager.GetByTarget(ctx, targetID2)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)
	s.Stop()
}

func TestP4RTConn_NonTLS(t *testing.T) {
	s := setup(t, getNonTLSServerConfig(t))

	connManager := NewConnManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestDestination(t, targetID1, deviceID1, true)

	conn, err := connManager.Connect(ctx, target1)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	capResponse, err := conn.Capabilities(ctx, &p4api.CapabilitiesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, capResponse.P4RuntimeApiVersion, "1.0.0")
	s.Stop()
}

func TestClient_Capabilities(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	connManager := NewConnManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestDestination(t, targetID1, deviceID1, true)
	conn, err := connManager.Connect(ctx, target1)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	capResponse, err := conn.Capabilities(ctx, &p4api.CapabilitiesRequest{})
	assert.NoError(t, err)
	assert.Equal(t, capResponse.P4RuntimeApiVersion, "1.0.0")
	s.Stop()
}

func TestClient_Write(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))
	connManager := NewConnManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	target1 := createTestDestination(t, targetID1, deviceID1, true)

	conn, err := connManager.Connect(ctx, target1)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	writeRequest := &p4api.WriteRequest{
		DeviceId: deviceID1,
	}
	writeResponse, err := conn.Write(ctx, writeRequest)
	assert.NoError(t, err)
	assert.NotNil(t, writeResponse)
	s.Stop()

}

func TestConnManager_Watch(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	connManager := NewConnManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan Conn)
	err := connManager.Watch(ctx, ch)
	assert.NoError(t, err)

	target1 := createTestDestination(t, targetID1, deviceID1, true)

	conn, err := connManager.Connect(ctx, target1)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	connEvent := <-ch
	assert.NotNil(t, connEvent)
	_, exist := connManager.Get(ctx, connEvent.ID())
	assert.Equal(t, true, exist)
	s.Stop()

}

func TestClient_SetForwardingPipelineConfig(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	connManager := NewConnManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	target1 := createTestDestination(t, targetID1, deviceID1, true)

	conn, err := connManager.Connect(ctx, target1)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	_, err = conn.SetForwardingPipelineConfig(ctx, &p4api.SetForwardingPipelineConfigRequest{})
	assert.NoError(t, err)
	s.Stop()

}

func TestClient_SetMasterArbitration(t *testing.T) {
	s := setup(t, getTLSServerConfig(t))

	connManager := NewConnManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	target1 := createTestDestination(t, targetID1, deviceID1, true)
	target2 := createTestDestination(t, targetID2, deviceID2, true)

	_, err := connManager.Connect(ctx, target1)
	assert.NoError(t, err)

	_, err = connManager.Connect(ctx, target2)
	assert.NoError(t, err)

	conn1, err := connManager.GetByTarget(ctx, targetID1)
	assert.NoError(t, err)
	assert.NotNil(t, conn1)

	conn2, err := connManager.GetByTarget(ctx, targetID2)
	assert.NoError(t, err)
	assert.NotNil(t, conn2)

	assert.NoError(t, err)

	role := p4utils.NewStratumRole("test1", 0, []byte{}, false, true)
	resp, err := conn1.PerformMasterArbitration(role)
	assert.NoError(t, err)
	assert.Equal(t, "test1", resp.Arbitration.Role.Name)

	role2 := p4utils.NewStratumRole("test2", 0, []byte{}, false, true)
	resp, err = conn2.PerformMasterArbitration(role2)
	assert.Equal(t, "test2", resp.Arbitration.Role.Name)

	assert.NoError(t, err)
	s.Stop()
}

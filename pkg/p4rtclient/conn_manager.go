// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package p4rtclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/google/uuid"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	p4v1 "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"math"
	"sync"
	"time"
)

var log = logging.GetLogger()

const (
	defaultTimeout = 60 * time.Second
)

// ConnManager p4rt connection manager
type ConnManager interface {
	Get(ctx context.Context, connID ConnID) (Conn, bool)
	GetByTarget(ctx context.Context, targetID topoapi.ID) (Client, error)
	Connect(ctx context.Context, destination *Destination) (Client, error)
	Disconnect(ctx context.Context, targetID topoapi.ID) error
	Watch(ctx context.Context, ch chan<- Conn) error
}

// NewConnManager creates a new p4rt connection manager
func NewConnManager() ConnManager {
	mgr := &connManager{
		targets:  make(map[topoapi.ID]*client),
		conns:    make(map[ConnID]Conn),
		watchers: make(map[uuid.UUID]chan<- Conn),
		eventCh:  make(chan Conn),
	}
	go mgr.processEvents()
	return mgr
}

type connManager struct {
	targets    map[topoapi.ID]*client
	conns      map[ConnID]Conn
	connsMu    sync.RWMutex
	watchers   map[uuid.UUID]chan<- Conn
	watchersMu sync.RWMutex
	eventCh    chan Conn
}

// Get returns a connection based on a given ID
func (m *connManager) Get(ctx context.Context, connID ConnID) (Conn, bool) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conn, ok := m.conns[connID]
	return conn, ok
}

// GetByTarget returns a P4RT client based on a given target ID
func (m *connManager) GetByTarget(ctx context.Context, targetID topoapi.ID) (Client, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	if p4rtClient, ok := m.targets[targetID]; ok {
		return p4rtClient, nil
	}
	return nil, errors.NewNotFound("p4rt client for target %s not found", targetID)
}

// Connect makes a gRPC connection to the target
func (m *connManager) Connect(ctx context.Context, destination *Destination) (Client, error) {
	m.connsMu.RLock()
	targetID := destination.TargetID
	if targetID == "" {
		targetID = topoapi.ID(fmt.Sprintf(destination.Endpoint.Address, ":", destination.Endpoint.Port))
	}
	if destination.Timeout == 0 {
		destination.Timeout = defaultTimeout
	}

	p4rtClient, ok := m.targets[targetID]
	m.connsMu.RUnlock()
	if ok {
		return nil, errors.NewAlreadyExists("target '%s' already exists", targetID)
	}

	m.connsMu.Lock()
	defer m.connsMu.Unlock()

	p4rtClient, ok = m.targets[targetID]
	if ok {
		return nil, errors.NewAlreadyExists("target '%s' already exists", targetID)
	}

	tlsOptions := destination.TLS
	tlsConfig := &tls.Config{}
	addr := fmt.Sprintf("%s:%d", destination.Endpoint.Address, destination.Endpoint.Port)
	if tlsOptions != nil {
		if tlsOptions.Plain {
			log.Infow("Plain (non TLS) connection to", "P4 runtime server address", addr)
		} else {
			if tlsOptions.Insecure {
				log.Infow("Insecure TLS connection to ", "P4 runtime server address", addr)
				tlsConfig = &tls.Config{InsecureSkipVerify: true}
			} else {
				log.Infow("Secure TLS connection to ", "P4 runtime server address", addr)
			}
			if tlsOptions.CaCert == "" {
				log.Info("Loading default CA")
				defaultCertPool, err := certs.GetCertPoolDefault()
				if err != nil {
					return nil, err
				}
				tlsConfig.RootCAs = defaultCertPool
			} else {
				certPool, err := certs.GetCertPool(tlsOptions.CaCert)
				if err != nil {
					return nil, err
				}
				tlsConfig.RootCAs = certPool
			}
			if tlsOptions.Cert == "" && tlsOptions.Key == "" {
				log.Info("Loading default certificates")
				clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
				if err != nil {
					return nil, err
				}
				tlsConfig.Certificates = []tls.Certificate{clientCerts}
			} else if tlsOptions.Cert != "" && tlsOptions.Key != "" {
				// Load certs given for device
				tlsCerts, err := setCertificate(tlsOptions.Cert, tlsOptions.Key)
				if err != nil {
					return nil, errors.NewInvalid(err.Error())
				}
				tlsConfig.Certificates = []tls.Certificate{tlsCerts}
			} else {
				log.Errorw("Can't load Ca=%s , Cert=%s , key=%s for %v, trying with insecure connection",
					"CA", tlsOptions.CaCert, "Cert", tlsOptions.Cert, "Key", tlsOptions.Key, "P4 runtime server address", addr)
				tlsConfig = &tls.Config{InsecureSkipVerify: true}
			}
		}
	}

	log.Infow("Connecting to P4RT target", "target ID", destination.TargetID)
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
	}
	p4rtClient, clientConn, err := connect(ctx, *destination, tlsConfig, opts...)
	if err != nil {
		log.Warnw("Failed to connect to the P4RT target %s: %s", "target ID", destination.TargetID, "error", err)
		return nil, err
	}

	m.targets[targetID] = p4rtClient
	streamChannel, err := p4rtClient.p4runtimeClient.StreamChannel(context.Background())
	if err != nil {
		log.Errorw("Cannot open a p4rt stream for connection", "targetID", destination.TargetID, "error", err)
		return nil, err
	}
	p4rtClient.streamClient.streamChannel = streamChannel
	go func() {
		var conn Conn
		state := clientConn.GetState()
		switch state {
		case connectivity.Ready:
			conn = newConn(targetID, p4rtClient, destination.DeviceID, destination.RoleName)
			m.addConn(conn)
		}

		for clientConn.WaitForStateChange(context.Background(), state) {
			state = clientConn.GetState()
			log.Infow("Connection state changed for Target", "target ID", targetID, "state", state)

			// If the channel is active, ensure a connection is added to the manager.
			// If the channel is idle, do not change its state (this can occur when
			// connected or disconnected).
			// In all other states, remove the connection from the manager.
			switch state {
			case connectivity.Ready:
				if conn == nil {
					conn = newConn(targetID, p4rtClient, destination.DeviceID, destination.RoleName)
					if err != nil {
						log.Warnw("Cannot open a p4rt stream for connection", "targetID", targetID, "error", err)
						continue
					}
					m.addConn(conn)
				}

			case connectivity.Idle:
				clientConn.Connect()
			default:
				if conn != nil {
					log.Infow("Removing connection ", "connectionID", conn.ID())
					m.removeConn(conn.ID())
					conn = nil
				}
			}

			// If the channel is shutting down, exit the goroutine.
			switch state {
			case connectivity.Shutdown:
				return
			}
		}
	}()

	return p4rtClient, nil
}

// Disconnect disconnects a gRPC connection based on a given target ID
func (m *connManager) Disconnect(ctx context.Context, targetID topoapi.ID) error {
	m.connsMu.Lock()
	clientConn, ok := m.targets[targetID]
	if !ok {
		m.connsMu.Unlock()
		return errors.NewNotFound("target '%s' not found", targetID)
	}
	delete(m.targets, targetID)
	m.connsMu.Unlock()
	return clientConn.Close()
}

func (m *connManager) Watch(ctx context.Context, ch chan<- Conn) error {
	id := uuid.New()
	m.watchersMu.Lock()
	m.connsMu.Lock()
	m.watchers[id] = ch
	m.watchersMu.Unlock()

	go func() {
		for _, conn := range m.conns {
			ch <- conn
		}
		m.connsMu.Unlock()

		<-ctx.Done()
		m.watchersMu.Lock()
		delete(m.watchers, id)
		m.watchersMu.Unlock()
	}()
	return nil
}

func (m *connManager) addConn(conn Conn) {
	m.connsMu.Lock()
	m.conns[conn.ID()] = conn
	m.connsMu.Unlock()
	m.eventCh <- conn
}

func (m *connManager) removeConn(connID ConnID) {
	m.connsMu.Lock()
	if conn, ok := m.conns[connID]; ok {
		delete(m.conns, connID)
		m.connsMu.Unlock()
		m.eventCh <- conn
	} else {
		m.connsMu.Unlock()
	}
}

func (m *connManager) processEvents() {
	for conn := range m.eventCh {
		m.processEvent(conn)
	}
}

func (m *connManager) processEvent(conn Conn) {
	log.Infow("Notifying P4RT connection", "connection ID", conn.ID())
	m.watchersMu.RLock()
	for _, watcher := range m.watchers {
		watcher <- conn
	}
	m.watchersMu.RUnlock()
}

func connect(ctx context.Context, d Destination, tlsConfig *tls.Config, opts ...grpc.DialOption) (*client, *grpc.ClientConn, error) {
	switch d.TLS {
	case nil:
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	default:
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	// TODO handle credentials

	gCtx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", d.Endpoint.Address, d.Endpoint.Port)
	conn, err := grpc.DialContext(gCtx, addr, opts...)
	if err != nil {
		return nil, nil, errors.NewInternal("Dialer(%s, %v): %v", addr, d.Timeout, err)
	}

	cl := p4v1.NewP4RuntimeClient(conn)

	p4rtClient := &client{
		grpcClient:      conn,
		p4runtimeClient: cl,
		streamClient: &streamClient{
			p4runtimeClient: cl,
			deviceID:        d.DeviceID,
		},
		deviceID: d.DeviceID,
	}

	return p4rtClient, conn, nil
}

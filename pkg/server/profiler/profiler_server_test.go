package profiler

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPprofProfiler(t *testing.T) {
	server := SingletonPprofServer()
	server.Start()

	// Test server is reachable
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", "6060"), 60*time.Second)
	require.NoError(t, err)
	if conn != nil {
		require.NoError(t, conn.Close())
	}

	// Test server was stopped
	server.Stop()
	conn, err = net.DialTimeout("tcp", net.JoinHostPort("localhost", "6060"), 2*time.Second)
	require.Error(t, err)
	require.Nil(t, conn)
}

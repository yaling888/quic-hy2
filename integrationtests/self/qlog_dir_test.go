package self_test

import (
	"context"
	"os"
	"path"
	"regexp"
	"testing"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/qlog"

	"github.com/stretchr/testify/require"
)

func TestQlogDirEnvironmentVariable(t *testing.T) {
	originalQlogDirValue := os.Getenv("QLOGDIR")
	tempTestDirPath, err := os.MkdirTemp("", "temp_test_dir")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Setenv("QLOGDIR", originalQlogDirValue))
		require.NoError(t, os.RemoveAll(tempTestDirPath))
	})
	qlogDir := path.Join(tempTestDirPath, "qlogs")
	require.NoError(t, os.Setenv("QLOGDIR", qlogDir))

	serverStopped := make(chan struct{})
	server, err := quic.ListenAddr(
		"localhost:0",
		getTLSConfig(),
		&quic.Config{
			Tracer: qlog.DefaultConnectionTracer,
		},
	)
	require.NoError(t, err)

	go func() {
		defer close(serverStopped)
		for {
			if _, err := server.Accept(context.Background()); err != nil {
				return
			}
		}
	}()

	conn, err := quic.DialAddr(
		context.Background(),
		server.Addr().String(),
		getTLSClientConfig(),
		&quic.Config{
			Tracer: qlog.DefaultConnectionTracer,
		},
	)
	require.NoError(t, err)
	conn.CloseWithError(0, "")
	server.Close()
	<-serverStopped

	_, err = os.Stat(tempTestDirPath)
	qlogDirCreated := !os.IsNotExist(err)
	require.True(t, qlogDirCreated)

	childs, err := os.ReadDir(qlogDir)
	require.NoError(t, err)
	require.Len(t, childs, 2)

	odcids := make([]string, 0, 2)
	vantagePoints := make([]string, 0, 2)
	qlogFileNameRegexp := regexp.MustCompile(`^([0-f]+)_(client|server).sqlog$`)

	for _, child := range childs {
		matches := qlogFileNameRegexp.FindStringSubmatch(child.Name())
		require.Len(t, matches, 3)
		odcids = append(odcids, matches[1])
		vantagePoints = append(vantagePoints, matches[2])
	}

	require.Equal(t, odcids[0], odcids[1])
	require.Contains(t, vantagePoints, "client")
	require.Contains(t, vantagePoints, "server")
}

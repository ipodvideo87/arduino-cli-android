package transport

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/stretchr/testify/require"
)

type scriptedRWC struct {
	readData  []byte
	readErr   error
	writeErr  error
	closed    bool
	closeErr  error
	writes    bytes.Buffer
	readCount int
}

func (s *scriptedRWC) Read(p []byte) (int, error) {
	s.readCount++
	if len(s.readData) == 0 {
		if s.readErr != nil {
			return 0, s.readErr
		}
		return 0, io.EOF
	}
	n := copy(p, s.readData)
	s.readData = s.readData[n:]
	if len(s.readData) == 0 && s.readErr != nil {
		return n, s.readErr
	}
	return n, nil
}

func (s *scriptedRWC) Write(p []byte) (int, error) {
	if s.writeErr != nil {
		return 0, s.writeErr
	}
	return s.writes.Write(p)
}

func (s *scriptedRWC) Close() error {
	s.closed = true
	return s.closeErr
}

func TestBoundedTransportStreamTracksLifecycleAndBytes(t *testing.T) {
	rwc := &scriptedRWC{readData: []byte("hello")}
	stream := NewBoundedTransportStream(rwc, TransportStreamDiagnosticsReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
		Beginner:      "stream foundation test",
	}, TransportStreamOptions{
		State:  TransportStreamStateReady,
		Bounds: TransportStreamBounds{MaxReadBytes: 3, MaxWriteBytes: 4},
	})
	require.NotNil(t, stream)

	buf := make([]byte, 8)
	n, err := stream.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 3, n)

	n, err = stream.Write([]byte("abcdef"))
	require.NoError(t, err)
	require.Equal(t, 4, n)

	require.NoError(t, stream.Close())

	report := stream.Diagnostics()
	require.Equal(t, TransportStreamStateClosed, report.State)
	require.Equal(t, int64(3), report.BytesRead)
	require.Equal(t, int64(4), report.BytesWritten)
	require.Equal(t, "closed by caller", report.CloseReason)
	require.NotZero(t, report.LastActivity)
	require.True(t, rwc.closed)
}

func TestBoundedTransportStreamEnforcesReadAndWriteBounds(t *testing.T) {
	rwc := &scriptedRWC{readData: []byte("abcdef")}
	stream := NewBoundedTransportStream(rwc, TransportStreamDiagnosticsReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
		Beginner:      "bounded stream boundary test",
	}, TransportStreamOptions{
		State:  TransportStreamStateExperimental,
		Bounds: TransportStreamBounds{MaxReadBytes: 2, MaxWriteBytes: 3},
	})
	require.NotNil(t, stream)

	buf := make([]byte, 8)
	n, err := stream.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	n, err = stream.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 0, n)

	n, err = stream.Write([]byte("abcd"))
	require.NoError(t, err)
	require.Equal(t, 3, n)

	n, err = stream.Write([]byte("zz"))
	require.ErrorIs(t, err, io.ErrShortWrite)
	require.Equal(t, 0, n)

	report := stream.Diagnostics()
	require.Equal(t, TransportStreamStateClosed, report.State)
	require.Equal(t, "write limit reached", report.CloseReason)
	require.Equal(t, int64(2), report.BytesRead)
	require.Equal(t, int64(3), report.BytesWritten)
}

func TestBoundedTransportStreamMarksEOFAndDisconnect(t *testing.T) {
	rwc := &scriptedRWC{readData: []byte("x")}
	stream := NewBoundedTransportStream(rwc, TransportStreamDiagnosticsReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
	}, TransportStreamOptions{
		State: TransportStreamStateReady,
	})
	require.NotNil(t, stream)

	buf := make([]byte, 1)
	n, err := stream.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	n, err = stream.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 0, n)

	report := stream.Diagnostics()
	require.Equal(t, TransportStreamStateDisconnected, report.State)
	require.Equal(t, "read returned EOF", report.DisconnectReason)
}

func TestBoundedTransportStreamSupportsTimeoutConfigurationAndCancel(t *testing.T) {
	rwc := &scriptedRWC{readData: []byte("data")}
	stream := NewBoundedTransportStream(rwc, TransportStreamDiagnosticsReport{
		SchemaVersion: "1",
		Status:        diagnostics.StatusWarning,
	}, TransportStreamOptions{
		State: TransportStreamStateExperimental,
	})
	require.NotNil(t, stream)

	timeoutController, ok := stream.(TransportStreamTimeoutController)
	require.True(t, ok)
	require.NoError(t, timeoutController.SetTimeouts(TransportStreamTimeouts{
		Read:  10 * time.Second,
		Write: 5 * time.Second,
		Idle:  15 * time.Second,
	}))
	require.Equal(t, 10*time.Second, timeoutController.Timeouts().Read)

	cancelable, ok := stream.(TransportStreamCancellationController)
	require.True(t, ok)
	require.NoError(t, cancelable.Cancel("user cancelled"))

	report := stream.Diagnostics()
	require.Equal(t, TransportStreamStateClosed, report.State)
	require.Equal(t, "user cancelled", report.CloseReason)
	require.Contains(t, report.Limitations[0], "stream deadline support is unavailable")
}

func TestStreamStateFromDiagnostics(t *testing.T) {
	require.Equal(t, TransportStreamStateReady, StreamStateFromDiagnostics(TransportStreamDiagnosticsReport{
		Status: diagnostics.StatusPassed,
	}))
	require.Equal(t, TransportStreamStateExperimental, StreamStateFromDiagnostics(TransportStreamDiagnosticsReport{
		Status: diagnostics.StatusWarning,
	}))
	require.Equal(t, TransportStreamStateFailed, StreamStateFromDiagnostics(TransportStreamDiagnosticsReport{
		Status: diagnostics.StatusFailed,
	}))
	require.Equal(t, TransportStreamStateUnavailable, StreamStateFromDiagnostics(TransportStreamDiagnosticsReport{
		Status: diagnostics.StatusSkipped,
	}))
	require.Equal(t, TransportStreamStateReady, StreamStateFromDiagnostics(TransportStreamDiagnosticsReport{
		State:  TransportStreamStateReady,
		Status: diagnostics.StatusFailed,
	}))
}

func TestDisconnectErrorClassification(t *testing.T) {
	require.True(t, isDisconnectError(errors.New("No such device")))
	require.True(t, isDisconnectError(errors.New("input/output error")))
	require.False(t, isDisconnectError(errors.New("permission denied")))
}

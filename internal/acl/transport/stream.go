package transport

import (
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
)

type TransportStreamBounds struct {
	MaxReadBytes  int64 `json:"max_read_bytes,omitempty"`
	MaxWriteBytes int64 `json:"max_write_bytes,omitempty"`
}

type TransportStreamOptions struct {
	State        TransportStreamState
	Bounds       TransportStreamBounds
	Timeouts     TransportStreamTimeouts
	Experimental bool
}

type deadlineReader interface {
	SetReadDeadline(time.Time) error
}

type deadlineWriter interface {
	SetWriteDeadline(time.Time) error
}

type deadlineRWC interface {
	io.ReadWriteCloser
	deadlineReader
	deadlineWriter
}

type boundedTransportStream struct {
	mu                sync.Mutex
	rwc               io.ReadWriteCloser
	state             TransportStreamState
	stateReason       string
	caps              TransportStreamCapabilities
	timeouts          TransportStreamTimeouts
	bounds            TransportStreamBounds
	bytesRead         int64
	bytesWritten      int64
	lastActivity      time.Time
	closeReason       string
	disconnectReason  string
	closed            bool
	eof               bool
	diagnosticsReport TransportStreamDiagnosticsReport
}

func NewBoundedTransportStream(rwc io.ReadWriteCloser, report TransportStreamDiagnosticsReport, opts TransportStreamOptions) TransportStream {
	if rwc == nil {
		return nil
	}
	s := &boundedTransportStream{
		rwc:               rwc,
		state:             opts.State,
		stateReason:       stateReasonFor(opts.State, opts.Experimental),
		caps:              transportStreamCapabilitiesFromOptions(opts),
		timeouts:          opts.Timeouts,
		bounds:            opts.Bounds,
		diagnosticsReport: report,
	}
	if s.state == "" {
		s.state = TransportStreamStateExperimental
	}
	if s.diagnosticsReport.SchemaVersion == "" {
		s.diagnosticsReport.SchemaVersion = "1"
	}
	if s.diagnosticsReport.State == "" {
		s.diagnosticsReport.State = s.state
	}
	if s.diagnosticsReport.StateReason == "" {
		s.diagnosticsReport.StateReason = s.stateReason
	}
	if s.diagnosticsReport.Status == "" {
		s.diagnosticsReport.Status = diagnostics.StatusWarning
	}
	s.refreshReportLocked()
	return s
}

func transportStreamCapabilitiesFromOptions(opts TransportStreamOptions) TransportStreamCapabilities {
	caps := TransportStreamCapabilities{
		Read:         true,
		Write:        true,
		Close:        true,
		Timeouts:     true,
		Cancellation: true,
		LastActivity: true,
		CloseReason:  true,
		EOF:          true,
		Disconnect:   true,
		Experimental: opts.Experimental || opts.State == TransportStreamStateExperimental,
	}
	return caps
}

func (s *boundedTransportStream) Read(p []byte) (int, error) {
	s.mu.Lock()
	if err := s.ensureReadableLocked(); err != nil {
		s.mu.Unlock()
		return 0, err
	}
	if s.bounds.MaxReadBytes > 0 {
		remaining := s.bounds.MaxReadBytes - s.bytesRead
		if remaining <= 0 {
			s.eof = true
			s.state = TransportStreamStateClosed
			s.closeReason = "read limit reached"
			s.refreshReportLocked()
			s.mu.Unlock()
			return 0, io.EOF
		}
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
	}
	s.setReadDeadlineLocked()
	rwc := s.rwc
	s.mu.Unlock()

	n, err := rwc.Read(p)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.bytesRead += int64(n)
	s.touchLocked()
	if n > 0 && s.state == TransportStreamStateReady {
		s.state = TransportStreamStateActive
	}
	switch {
	case err == nil:
	case errors.Is(err, io.EOF):
		s.eof = true
		if s.closed {
			s.state = TransportStreamStateClosed
			if s.closeReason == "" {
				s.closeReason = "eof after close"
			}
		} else {
			s.state = TransportStreamStateDisconnected
			s.disconnectReason = "read returned EOF"
		}
	default:
		if isDisconnectError(err) {
			s.state = TransportStreamStateDisconnected
			s.disconnectReason = err.Error()
		} else if !s.closed {
			s.state = TransportStreamStateFailed
			s.disconnectReason = err.Error()
		}
	}
	s.refreshReportLocked()
	return n, err
}

func (s *boundedTransportStream) Write(p []byte) (int, error) {
	s.mu.Lock()
	if err := s.ensureWritableLocked(); err != nil {
		s.mu.Unlock()
		return 0, err
	}
	if s.bounds.MaxWriteBytes > 0 {
		remaining := s.bounds.MaxWriteBytes - s.bytesWritten
		if remaining <= 0 {
			s.state = TransportStreamStateClosed
			s.closeReason = "write limit reached"
			s.refreshReportLocked()
			s.mu.Unlock()
			return 0, io.ErrShortWrite
		}
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
	}
	s.setWriteDeadlineLocked()
	rwc := s.rwc
	s.mu.Unlock()

	n, err := rwc.Write(p)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.bytesWritten += int64(n)
	s.touchLocked()
	if n > 0 && s.state == TransportStreamStateReady {
		s.state = TransportStreamStateActive
	}
	switch {
	case err == nil:
	case errors.Is(err, io.EOF):
		s.eof = true
		if s.closed {
			s.state = TransportStreamStateClosed
		} else {
			s.state = TransportStreamStateDisconnected
			s.disconnectReason = "write returned EOF"
		}
	default:
		if isDisconnectError(err) {
			s.state = TransportStreamStateDisconnected
			s.disconnectReason = err.Error()
		} else if !s.closed {
			s.state = TransportStreamStateFailed
			s.disconnectReason = err.Error()
		}
	}
	s.refreshReportLocked()
	return n, err
}

func (s *boundedTransportStream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.state = TransportStreamStateClosing
	s.touchLocked()
	rwc := s.rwc
	s.mu.Unlock()

	err := rwc.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.touchLocked()
	if err != nil {
		s.state = TransportStreamStateFailed
		s.closeReason = err.Error()
		if s.disconnectReason == "" {
			s.disconnectReason = err.Error()
		}
	} else {
		s.state = TransportStreamStateClosed
		if s.closeReason == "" {
			s.closeReason = "closed by caller"
		}
	}
	s.refreshReportLocked()
	return err
}

func (s *boundedTransportStream) Capabilities() TransportStreamCapabilities {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.caps
}

func (s *boundedTransportStream) Diagnostics() TransportStreamDiagnosticsReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.diagnosticsReport
}

func (s *boundedTransportStream) State() TransportStreamState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *boundedTransportStream) Timeouts() TransportStreamTimeouts {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.timeouts
}

func (s *boundedTransportStream) SetTimeouts(timeouts TransportStreamTimeouts) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeouts = timeouts
	if s.state == "" {
		s.state = TransportStreamStateExperimental
	}
	if !s.supportsDeadlinesLocked() && (timeouts.Read != 0 || timeouts.Write != 0 || timeouts.Idle != 0) {
		s.diagnosticsReport.Limitations = appendUniqueString(s.diagnosticsReport.Limitations, "stream deadline support is unavailable on this underlying transport")
	}
	s.refreshReportLocked()
	return nil
}

func (s *boundedTransportStream) Cancel(reason string) error {
	s.mu.Lock()
	if reason != "" {
		s.closeReason = reason
	}
	s.state = TransportStreamStateClosing
	s.mu.Unlock()
	return s.Close()
}

func (s *boundedTransportStream) LastActivity() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastActivity
}

func (s *boundedTransportStream) EOF() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.eof
}

func (s *boundedTransportStream) Disconnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == TransportStreamStateDisconnected
}

func (s *boundedTransportStream) DisconnectReason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.disconnectReason
}

func (s *boundedTransportStream) CloseReason() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeReason
}

func (s *boundedTransportStream) ensureReadableLocked() error {
	if s.closed {
		return io.ErrClosedPipe
	}
	if s.state == TransportStreamStateFailed || s.state == TransportStreamStateDisconnected {
		if s.disconnectReason != "" {
			return errors.New(s.disconnectReason)
		}
		return io.ErrUnexpectedEOF
	}
	if s.state == "" {
		s.state = TransportStreamStateExperimental
	}
	return nil
}

func (s *boundedTransportStream) ensureWritableLocked() error {
	if s.closed {
		return io.ErrClosedPipe
	}
	if s.state == TransportStreamStateFailed || s.state == TransportStreamStateDisconnected {
		if s.disconnectReason != "" {
			return errors.New(s.disconnectReason)
		}
		return io.ErrClosedPipe
	}
	if s.state == "" {
		s.state = TransportStreamStateExperimental
	}
	return nil
}

func (s *boundedTransportStream) supportsDeadlinesLocked() bool {
	_, readOK := s.rwc.(deadlineReader)
	_, writeOK := s.rwc.(deadlineWriter)
	return readOK || writeOK
}

func (s *boundedTransportStream) setReadDeadlineLocked() {
	if s.timeouts.Read == 0 && s.timeouts.Idle == 0 {
		return
	}
	if deadline, ok := s.rwc.(deadlineReader); ok {
		_ = deadline.SetReadDeadline(time.Now().Add(maxDuration(s.timeouts.Read, s.timeouts.Idle)))
	}
}

func (s *boundedTransportStream) setWriteDeadlineLocked() {
	if s.timeouts.Write == 0 {
		return
	}
	if deadline, ok := s.rwc.(deadlineWriter); ok {
		_ = deadline.SetWriteDeadline(time.Now().Add(s.timeouts.Write))
	}
}

func (s *boundedTransportStream) touchLocked() {
	s.lastActivity = time.Now()
}

func (s *boundedTransportStream) refreshReportLocked() {
	s.diagnosticsReport.State = s.state
	s.diagnosticsReport.Timeouts = s.timeouts
	s.diagnosticsReport.BytesRead = s.bytesRead
	s.diagnosticsReport.BytesWritten = s.bytesWritten
	s.diagnosticsReport.LastActivity = s.lastActivity
	s.diagnosticsReport.CloseReason = s.closeReason
	s.diagnosticsReport.DisconnectReason = s.disconnectReason
	switch s.state {
	case TransportStreamStateDisconnected:
		if s.disconnectReason != "" {
			s.diagnosticsReport.StateReason = s.disconnectReason
		}
	case TransportStreamStateClosed:
		if s.closeReason != "" {
			s.diagnosticsReport.StateReason = s.closeReason
		}
	case TransportStreamStateFailed:
		if s.disconnectReason != "" {
			s.diagnosticsReport.StateReason = s.disconnectReason
		}
	case TransportStreamStateExperimental:
		if s.stateReason != "" {
			s.diagnosticsReport.StateReason = s.stateReason
		}
	}
	if s.state == TransportStreamStateReady || s.state == TransportStreamStateActive {
		s.diagnosticsReport.StreamSupported = true
	}
	if s.state == TransportStreamStateExperimental {
		s.diagnosticsReport.StreamSupported = false
		s.diagnosticsReport.StreamProven = false
	}
	if s.state == TransportStreamStateDisconnected {
		s.diagnosticsReport.DisconnectState = StreamObservationFailed
	}
	if s.state == TransportStreamStateClosed {
		s.diagnosticsReport.CloseState = StreamObservationPassed
	}
}

func isDisconnectError(err error) bool {
	if err == nil {
		return false
	}
	joined := strings.ToLower(err.Error())
	for _, needle := range []string{
		"no such device",
		"bad file descriptor",
		"device not configured",
		"input/output error",
		"broken pipe",
	} {
		if strings.Contains(joined, needle) {
			return true
		}
	}
	return false
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func stateReasonFor(state TransportStreamState, experimental bool) string {
	switch state {
	case TransportStreamStateExperimental:
		return "experimental stream bridge"
	case TransportStreamStateReady:
		return "stream ready"
	case TransportStreamStateActive:
		return "stream active"
	case TransportStreamStateClosing:
		return "stream closing"
	case TransportStreamStateClosed:
		return "stream closed"
	case TransportStreamStateDisconnected:
		return "stream disconnected"
	case TransportStreamStateFailed:
		return "stream failed"
	case TransportStreamStateUnavailable:
		return "stream unavailable"
	default:
		if experimental {
			return "experimental stream bridge"
		}
		return ""
	}
}

func NewDiagnosticStream(rwc io.ReadWriteCloser, report TransportStreamDiagnosticsReport) TransportStream {
	return NewBoundedTransportStream(rwc, report, TransportStreamOptions{
		State:        TransportStreamStateExperimental,
		Experimental: true,
	})
}

func NewExperimentalTransportStream(rwc io.ReadWriteCloser, report TransportStreamDiagnosticsReport) TransportStream {
	return NewBoundedTransportStream(rwc, report, TransportStreamOptions{
		State:        TransportStreamStateExperimental,
		Experimental: true,
	})
}

func StreamStateFromDiagnostics(report TransportStreamDiagnosticsReport) TransportStreamState {
	if report.State != "" {
		return report.State
	}
	switch report.Status {
	case diagnostics.StatusPassed:
		return TransportStreamStateReady
	case diagnostics.StatusWarning:
		return TransportStreamStateExperimental
	case diagnostics.StatusFailed:
		return TransportStreamStateFailed
	case diagnostics.StatusSkipped:
		return TransportStreamStateUnavailable
	default:
		return TransportStreamStateUnavailable
	}
}

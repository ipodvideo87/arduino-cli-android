package transport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	desc TransportDescriptor
}

func (f fakeProvider) Descriptor() TransportDescriptor { return f.desc }

func TestTransportManagerSelectsHighestPriorityMatchingTransport(t *testing.T) {
	mgr := NewTransportManager(
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindNativeSerial,
			Name:         "native",
			Available:    true,
			Priority:     100,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityLineCoding},
		}},
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindPTY,
			Name:         "pty",
			Available:    true,
			Priority:     80,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityPTYExport},
		}},
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindRFC2217,
			Name:         "rfc",
			Available:    true,
			Priority:     70,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityRFC2217},
		}},
	)

	selected, err := mgr.Select(SelectionRequest{RequiredCapabilities: []Capability{CapabilitySerialIO}})
	require.NoError(t, err)
	require.Equal(t, KindNativeSerial, selected.Descriptor.Kind)
	require.Len(t, selected.Alternatives, 2)
	require.Contains(t, selected.Reason, "serial-io")
}

func TestTransportManagerHonorsPreferredKinds(t *testing.T) {
	mgr := NewTransportManager(
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindNativeSerial,
			Name:         "native",
			Available:    true,
			Priority:     100,
			Capabilities: []Capability{CapabilitySerialIO},
		}},
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindPTY,
			Name:         "pty",
			Available:    true,
			Priority:     1,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityPTYExport},
		}},
	)

	selected, err := mgr.Select(SelectionRequest{
		RequiredCapabilities: []Capability{CapabilitySerialIO},
		PreferredKinds:       []Kind{KindPTY},
	})
	require.NoError(t, err)
	require.Equal(t, KindPTY, selected.Descriptor.Kind)
}

func TestTransportManagerSkipsUnavailableProvidersDuringSelection(t *testing.T) {
	mgr := NewTransportManager(
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindAndroidUSBFD,
			Name:         "unavailable",
			Available:    false,
			Priority:     200,
			Capabilities: []Capability{CapabilitySerialIO},
		}},
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindNativeSerial,
			Name:         "available",
			Available:    true,
			Priority:     100,
			Capabilities: []Capability{CapabilitySerialIO},
		}},
	)

	selected, err := mgr.Select(SelectionRequest{RequiredCapabilities: []Capability{CapabilitySerialIO}})
	require.NoError(t, err)
	require.Equal(t, "available", selected.Descriptor.Name)
}

func TestTransportManagerSelectsFutureTransportWhenRequested(t *testing.T) {
	mgr := NewTransportManager(
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindFuture,
			Name:         "custom",
			Available:    true,
			Priority:     200,
			Capabilities: []Capability{CapabilityFuture, CapabilitySerialIO},
		}},
	)

	selected, err := mgr.Select(SelectionRequest{RequiredCapabilities: []Capability{CapabilityFuture}})
	require.NoError(t, err)
	require.Equal(t, KindFuture, selected.Descriptor.Kind)
}

func TestTransportManagerAvailableReturnsSortedDescriptors(t *testing.T) {
	mgr := NewTransportManager(
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindRFC2217,
			Name:         "rfc",
			Available:    true,
			Priority:     10,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityRFC2217},
		}},
		fakeProvider{desc: TransportDescriptor{
			Kind:         KindPTY,
			Name:         "pty",
			Available:    true,
			Priority:     50,
			Capabilities: []Capability{CapabilitySerialIO, CapabilityPTYExport},
		}},
	)

	available := mgr.Available()
	require.Len(t, available, 2)
	require.Equal(t, KindPTY, available[0].Kind)
	require.Equal(t, KindRFC2217, available[1].Kind)
}

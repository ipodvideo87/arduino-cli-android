package transport

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Kind string

const (
	KindNativeSerial Kind = "native-serial"
	KindAndroidUSBFD Kind = "android-usb-fd"
	KindPTY          Kind = "pty"
	KindRFC2217      Kind = "rfc2217"
	KindFuture       Kind = "future"
)

type Capability string

const (
	CapabilitySerialIO            Capability = "serial-io"
	CapabilityDescriptorDiscovery Capability = "descriptor-discovery"
	CapabilityUSBHandle           Capability = "usb-handle"
	CapabilityPTYExport           Capability = "pty-export"
	CapabilityRFC2217             Capability = "rfc2217"
	CapabilityLineCoding          Capability = "line-coding"
	CapabilityModemControl        Capability = "modem-control"
	CapabilityFuture              Capability = "future"
)

type TransportDescriptor struct {
	Kind         Kind              `json:"kind"`
	Name         string            `json:"name"`
	Available    bool              `json:"available"`
	Priority     int               `json:"priority"`
	Capabilities []Capability      `json:"capabilities,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type TransportProvider interface {
	Descriptor() TransportDescriptor
}

type SelectionRequest struct {
	RequiredCapabilities []Capability `json:"required_capabilities,omitempty"`
	PreferredKinds       []Kind       `json:"preferred_kinds,omitempty"`
}

type TransportSelection struct {
	Descriptor   TransportDescriptor   `json:"descriptor"`
	Alternatives []TransportDescriptor `json:"alternatives,omitempty"`
	Reason       string                `json:"reason,omitempty"`
}

type TransportManager struct {
	providers map[Kind]TransportProvider
}

func NewTransportManager(providers ...TransportProvider) *TransportManager {
	m := &TransportManager{providers: map[Kind]TransportProvider{}}
	for _, provider := range providers {
		m.Register(provider)
	}
	return m
}

func (m *TransportManager) Register(provider TransportProvider) {
	if m.providers == nil {
		m.providers = map[Kind]TransportProvider{}
	}
	if provider == nil {
		return
	}
	desc := provider.Descriptor()
	if desc.Kind == "" {
		return
	}
	m.providers[desc.Kind] = provider
}

func (m *TransportManager) Available() []TransportDescriptor {
	descriptors := make([]TransportDescriptor, 0, len(m.providers))
	for _, provider := range m.providers {
		desc := normalizeDescriptor(provider.Descriptor())
		if desc.Available {
			descriptors = append(descriptors, desc)
		}
	}
	sort.SliceStable(descriptors, func(i, j int) bool {
		if descriptors[i].Priority != descriptors[j].Priority {
			return descriptors[i].Priority > descriptors[j].Priority
		}
		if descriptors[i].Kind != descriptors[j].Kind {
			return descriptors[i].Kind < descriptors[j].Kind
		}
		return descriptors[i].Name < descriptors[j].Name
	})
	return descriptors
}

func (m *TransportManager) Select(req SelectionRequest) (TransportSelection, error) {
	descriptors := m.Available()
	matches := make([]scoredTransport, 0, len(descriptors))
	for _, desc := range descriptors {
		if !desc.Satisfies(req.RequiredCapabilities) {
			continue
		}
		matches = append(matches, scoredTransport{
			TransportDescriptor: desc,
			score:               scoreTransport(desc, req),
		})
	}
	if len(matches) == 0 {
		return TransportSelection{}, errors.New("no transport satisfies the requested capabilities")
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if matches[i].Priority != matches[j].Priority {
			return matches[i].Priority > matches[j].Priority
		}
		if matches[i].Kind != matches[j].Kind {
			return matches[i].Kind < matches[j].Kind
		}
		return matches[i].Name < matches[j].Name
	})

	selected := matches[0]
	alternatives := make([]TransportDescriptor, 0, len(matches)-1)
	for _, match := range matches[1:] {
		alternatives = append(alternatives, match.TransportDescriptor)
	}
	return TransportSelection{
		Descriptor:   selected.TransportDescriptor,
		Alternatives: alternatives,
		Reason:       fmt.Sprintf("selected %s for %s", selected.Kind, capabilitiesString(req.RequiredCapabilities)),
	}, nil
}

func (d TransportDescriptor) Satisfies(required []Capability) bool {
	if len(required) == 0 {
		return true
	}
	have := map[Capability]struct{}{}
	for _, capability := range d.Capabilities {
		have[capability] = struct{}{}
	}
	for _, capability := range required {
		if _, ok := have[capability]; !ok {
			return false
		}
	}
	return true
}

type scoredTransport struct {
	TransportDescriptor
	score int
}

func scoreTransport(desc TransportDescriptor, req SelectionRequest) int {
	score := desc.Priority * 100
	caps := map[Capability]struct{}{}
	for _, capability := range desc.Capabilities {
		caps[capability] = struct{}{}
		score += 2
	}
	for _, capability := range req.RequiredCapabilities {
		if _, ok := caps[capability]; ok {
			score += 50
		}
	}
	for i, preferred := range req.PreferredKinds {
		if preferred == desc.Kind {
			score += 1_000_000 - i
		}
	}
	return score
}

func normalizeDescriptor(desc TransportDescriptor) TransportDescriptor {
	desc.Name = strings.TrimSpace(desc.Name)
	if desc.Metadata == nil {
		desc.Metadata = map[string]string{}
	}
	sort.SliceStable(desc.Capabilities, func(i, j int) bool {
		return desc.Capabilities[i] < desc.Capabilities[j]
	})
	return desc
}

func capabilitiesString(caps []Capability) string {
	if len(caps) == 0 {
		return "<none>"
	}
	values := make([]string, 0, len(caps))
	for _, capability := range caps {
		values = append(values, string(capability))
	}
	sort.Strings(values)
	return strings.Join(values, ", ")
}

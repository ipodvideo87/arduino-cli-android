package transport

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
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
	CapabilityDiscovery           Capability = "discovery"
	CapabilityPermission          Capability = "permission"
	CapabilitySerialIO            Capability = "serial-io"
	CapabilityControlLines        Capability = "control-lines"
	CapabilityLineCoding          Capability = "line-coding"
	CapabilityResetSignal         Capability = "reset-signal"
	CapabilityBulkTransfer        Capability = "bulk-transfer"
	CapabilityInterruptTransfer   Capability = "interrupt-transfer"
	CapabilityControlTransfer     Capability = "control-transfer"
	CapabilityPTYExport           Capability = "pty-export"
	CapabilityRFC2217             Capability = "rfc2217"
	CapabilityUploadEndpoint      Capability = "upload-endpoint"
	CapabilityMonitorEndpoint     Capability = "monitor-endpoint"
	CapabilityUSBHandle           Capability = "usb-handle"
	CapabilityDescriptorDiscovery Capability = "descriptor-discovery"
	CapabilityModemControl        Capability = "modem-control"
	CapabilityFuture              Capability = "future"
)

type TransportDescriptor struct {
	SchemaVersion string                `json:"schema_version,omitempty"`
	Provider      string                `json:"provider,omitempty"`
	StableID      string                `json:"stable_id,omitempty"`
	Family        string                `json:"family,omitempty"`
	Kind          Kind                  `json:"kind"`
	Name          string                `json:"name"`
	Available     bool                  `json:"available"`
	Priority      int                   `json:"priority"`
	Capabilities  []Capability          `json:"capabilities,omitempty"`
	CapabilitySet TransportCapabilities `json:"capability_set,omitempty"`
	Metadata      map[string]string     `json:"metadata,omitempty"`
}

type TransportProvider interface {
	Descriptor() TransportDescriptor
}

type SelectionRequest struct {
	RequiredCapabilities []Capability `json:"required_capabilities,omitempty"`
	PreferredKinds       []Kind       `json:"preferred_kinds,omitempty"`
}

type TransportSelection struct {
	Provider     TransportProvider     `json:"-"`
	Descriptor   TransportDescriptor   `json:"descriptor"`
	Alternatives []TransportDescriptor `json:"alternatives,omitempty"`
	Reason       string                `json:"reason,omitempty"`
}

type providerEntry struct {
	provider   TransportProvider
	descriptor TransportDescriptor
}

type TransportManager struct {
	providers []TransportProvider
}

func NewTransportManager(providers ...TransportProvider) *TransportManager {
	m := &TransportManager{}
	for _, provider := range providers {
		m.Register(provider)
	}
	return m
}

func (m *TransportManager) Register(provider TransportProvider) {
	if provider == nil {
		return
	}
	m.providers = append(m.providers, provider)
}

func (m *TransportManager) Available() []TransportDescriptor {
	entries := m.providerEntries()
	descriptors := make([]TransportDescriptor, 0, len(entries))
	for _, entry := range entries {
		if entry.descriptor.Available {
			descriptors = append(descriptors, entry.descriptor)
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
	entries := m.providerEntries()
	matches := make([]scoredTransport, 0, len(entries))
	for _, entry := range entries {
		desc := entry.descriptor
		if !desc.Available {
			continue
		}
		if !desc.Satisfies(req.RequiredCapabilities) {
			continue
		}
		matches = append(matches, scoredTransport{
			TransportDescriptor: desc,
			provider:            entry.provider,
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
		Provider:     selected.provider,
		Descriptor:   selected.TransportDescriptor,
		Alternatives: alternatives,
		Reason:       fmt.Sprintf("selected %s for %s", selected.Kind, capabilitiesString(req.RequiredCapabilities)),
	}, nil
}

func (m *TransportManager) Discover(ctx context.Context, req DiscoveryRequest) ([]DiscoveredDevice, error) {
	devices := make([]DiscoveredDevice, 0)
	for _, entry := range m.providerEntries() {
		discoverer, ok := entry.provider.(TransportDiscoverer)
		if !ok {
			continue
		}
		found, err := discoverer.Discover(ctx, req)
		if err != nil {
			return nil, err
		}
		devices = append(devices, found...)
	}
	sort.SliceStable(devices, func(i, j int) bool {
		if devices[i].Provider != devices[j].Provider {
			return devices[i].Provider < devices[j].Provider
		}
		if devices[i].StableID != devices[j].StableID {
			return devices[i].StableID < devices[j].StableID
		}
		return devices[i].DisplayName < devices[j].DisplayName
	})
	return devices, nil
}

func (m *TransportManager) RequestPermission(ctx context.Context, req SelectionRequest, permission PermissionRequest) (PermissionResult, error) {
	selected, err := m.Select(req)
	if err != nil {
		return PermissionResult{
			State:       PermissionStateUnavailable,
			Reason:      err.Error(),
			UserMessage: "no transport is available for permission acquisition",
		}, err
	}

	permission.Selection = req
	if len(permission.RequiredCapabilities) == 0 {
		permission.RequiredCapabilities = append([]Capability(nil), req.RequiredCapabilities...)
	}
	if permission.Device.Provider == "" {
		permission.Device.Provider = descriptorIdentity(selected.Descriptor)
	}
	if permission.Device.TransportFamily == "" {
		permission.Device.TransportFamily = TransportFamily(selected.Descriptor.Family)
	}

	requester, ok := selected.Provider.(PermissionRequester)
	if !ok {
		return PermissionResult{
			State:         PermissionStateUnavailable,
			Reason:        "selected provider does not support permission acquisition",
			UserMessage:   "permission acquisition is unavailable for the selected transport",
			Professional:  []string{selected.Reason},
			Metadata:      permission.Metadata,
			Required:      permission.Required,
			Method:        permission.Method,
			RetryGuidance: permission.RetryGuidance,
		}, nil
	}

	result, err := requester.RequestPermission(ctx, permission)
	if result.State == "" {
		result.State = PermissionStateGranted
	}
	if result.Metadata == nil {
		result.Metadata = permission.Metadata
	}
	return result, err
}

func (m *TransportManager) Open(ctx context.Context, req SelectionRequest, open OpenRequest) (TransportSession, error) {
	selected, err := m.Select(req)
	if err != nil {
		return nil, err
	}
	open.Selection = req
	if len(open.RequiredCapabilities) == 0 {
		open.RequiredCapabilities = append([]Capability(nil), req.RequiredCapabilities...)
	}
	if open.Device.Provider == "" {
		open.Device.Provider = descriptorIdentity(selected.Descriptor)
	}
	opener, ok := selected.Provider.(SessionOpener)
	if !ok {
		return nil, errors.New("selected provider does not support session opening")
	}
	return opener.Open(ctx, open)
}

func (m *TransportManager) Diagnostics(ctx context.Context, req SelectionRequest) (TransportDiagnosticsReport, error) {
	selected, err := m.Select(req)
	if err != nil {
		return TransportDiagnosticsReport{
			SchemaVersion:   "1",
			Status:          diagnostics.StatusFailed,
			DiscoveryStatus: diagnostics.StatusFailed,
			Beginner:        "no transport satisfies the requested capabilities",
			Professional:    []string{err.Error()},
			Warnings:        []string{err.Error()},
			Fields:          map[string]string{"selection_error": err.Error()},
			Metadata:        map[string]string{},
		}, err
	}

	report := TransportDiagnosticsReport{
		SchemaVersion:    "1",
		Status:           diagnostics.StatusPassed,
		Provider:         descriptorIdentity(selected.Descriptor),
		ProviderKind:     selected.Descriptor.Kind,
		Selected:         selected.Descriptor,
		Alternatives:     append([]TransportDescriptor(nil), selected.Alternatives...),
		DiscoveryStatus:  diagnostics.StatusPassed,
		PermissionStatus: diagnostics.StatusSkipped,
		ConnectionStatus: diagnostics.StatusSkipped,
		Beginner:         fmt.Sprintf("selected %s", selected.Descriptor.Name),
		Professional:     []string{selected.Reason},
		Fields:           map[string]string{},
		Metadata:         map[string]string{},
	}
	report.Device = discoveredDeviceFromDescriptor(selected.Descriptor)
	report.Interfaces = append([]InterfaceSummary(nil), report.Device.Interfaces...)
	report.Endpoints = flattenEndpoints(report.Interfaces)
	report.Fields["selected_kind"] = string(selected.Descriptor.Kind)
	report.Fields["capabilities"] = capabilitiesString(req.RequiredCapabilities)
	report.Fields["alternatives"] = fmt.Sprintf("%d", len(selected.Alternatives))

	if reporter, ok := selected.Provider.(DiagnosticsReporter); ok {
		diag, diagErr := reporter.Diagnostics(ctx, DiagnosticsRequest{
			Selection: req,
			Selected:  selected,
			Device:    &report.Device,
		})
		if diagErr != nil {
			return TransportDiagnosticsReport{}, diagErr
		}
		report = mergeDiagnostics(report, diag)
	}
	return report, nil
}

type scoredTransport struct {
	TransportDescriptor
	provider TransportProvider
	score    int
}

func scoreTransport(desc TransportDescriptor, req SelectionRequest) int {
	score := desc.Priority * 100
	caps := map[Capability]struct{}{}
	for _, capability := range desc.Capabilities {
		caps[capability] = struct{}{}
		score += 2
	}
	for _, capability := range desc.CapabilitySet.List() {
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

func (d TransportDescriptor) Satisfies(required []Capability) bool {
	if len(required) == 0 {
		return true
	}
	have := map[Capability]struct{}{}
	for _, capability := range d.Capabilities {
		have[capability] = struct{}{}
	}
	for _, capability := range d.CapabilitySet.List() {
		have[capability] = struct{}{}
	}
	for _, capability := range required {
		if _, ok := have[capability]; !ok {
			return false
		}
	}
	return true
}

func normalizeDescriptor(desc TransportDescriptor) TransportDescriptor {
	desc.Name = strings.TrimSpace(desc.Name)
	desc.Provider = strings.TrimSpace(desc.Provider)
	desc.StableID = strings.TrimSpace(desc.StableID)
	desc.Family = strings.TrimSpace(desc.Family)
	if desc.Metadata == nil {
		desc.Metadata = map[string]string{}
	}
	sort.SliceStable(desc.Capabilities, func(i, j int) bool {
		return desc.Capabilities[i] < desc.Capabilities[j]
	})
	desc.CapabilitySet = CapabilitiesFromList(desc.Capabilities...)
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

func (m *TransportManager) providerEntries() []providerEntry {
	entries := make([]providerEntry, 0, len(m.providers))
	for _, provider := range m.providers {
		if provider == nil {
			continue
		}
		desc := normalizeDescriptor(provider.Descriptor())
		if desc.Kind == "" {
			continue
		}
		entries = append(entries, providerEntry{provider: provider, descriptor: desc})
	}
	return entries
}

func discoveredDeviceFromDescriptor(desc TransportDescriptor) DiscoveredDevice {
	return DiscoveredDevice{
		Provider:        descriptorIdentity(desc),
		DisplayName:     desc.Name,
		TransportFamily: TransportFamily(desc.Family),
		Capabilities:    desc.CapabilitySet,
		Metadata:        cloneStringMap(desc.Metadata),
	}
}

func flattenEndpoints(interfaces []InterfaceSummary) []EndpointSummary {
	var endpoints []EndpointSummary
	for _, iface := range interfaces {
		endpoints = append(endpoints, iface.Endpoints...)
	}
	return endpoints
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func descriptorIdentity(desc TransportDescriptor) string {
	if strings.TrimSpace(desc.Provider) != "" {
		return strings.TrimSpace(desc.Provider)
	}
	return strings.TrimSpace(desc.Name)
}

func mergeDiagnostics(base, overlay TransportDiagnosticsReport) TransportDiagnosticsReport {
	if overlay.SchemaVersion != "" {
		base.SchemaVersion = overlay.SchemaVersion
	}
	if overlay.Status != "" {
		base.Status = overlay.Status
	}
	if overlay.Provider != "" {
		base.Provider = overlay.Provider
	}
	if overlay.ProviderKind != "" {
		base.ProviderKind = overlay.ProviderKind
	}
	if overlay.Selected.Kind != "" {
		base.Selected = overlay.Selected
	}
	if len(overlay.Alternatives) > 0 {
		base.Alternatives = append([]TransportDescriptor(nil), overlay.Alternatives...)
	}
	if overlay.Device.DisplayName != "" || overlay.Device.Provider != "" || len(overlay.Device.Interfaces) > 0 {
		base.Device = overlay.Device
		if len(base.Interfaces) == 0 {
			base.Interfaces = append([]InterfaceSummary(nil), overlay.Device.Interfaces...)
		}
	}
	if len(overlay.Devices) > 0 {
		base.Devices = append([]DiscoveredDevice(nil), overlay.Devices...)
	}
	if overlay.DiscoveryStatus != "" {
		base.DiscoveryStatus = overlay.DiscoveryStatus
	}
	if overlay.PermissionStatus != "" {
		base.PermissionStatus = overlay.PermissionStatus
	}
	if overlay.ConnectionStatus != "" {
		base.ConnectionStatus = overlay.ConnectionStatus
	}
	if overlay.SelectedEndpoint.Kind != "" || overlay.SelectedEndpoint.Path != "" || overlay.SelectedEndpoint.URL != "" {
		base.SelectedEndpoint = overlay.SelectedEndpoint
	}
	if len(overlay.Interfaces) > 0 {
		base.Interfaces = append([]InterfaceSummary(nil), overlay.Interfaces...)
	}
	if len(overlay.Endpoints) > 0 {
		base.Endpoints = append([]EndpointSummary(nil), overlay.Endpoints...)
	}
	if len(overlay.Warnings) > 0 {
		base.Warnings = append([]string(nil), overlay.Warnings...)
	}
	if len(overlay.Limitations) > 0 {
		base.Limitations = append([]string(nil), overlay.Limitations...)
	}
	if strings.TrimSpace(overlay.Beginner) != "" {
		base.Beginner = overlay.Beginner
	}
	if len(overlay.Professional) > 0 {
		base.Professional = append([]string(nil), overlay.Professional...)
	}
	if len(overlay.Fields) > 0 {
		if base.Fields == nil {
			base.Fields = map[string]string{}
		}
		for key, value := range overlay.Fields {
			base.Fields[key] = value
		}
	}
	if len(overlay.Metadata) > 0 {
		if base.Metadata == nil {
			base.Metadata = map[string]string{}
		}
		for key, value := range overlay.Metadata {
			base.Metadata[key] = value
		}
	}
	return base
}

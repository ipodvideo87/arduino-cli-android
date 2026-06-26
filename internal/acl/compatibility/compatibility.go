package compatibility

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	semver "go.bug.st/relaxed-semver"
)

type Scope string

const (
	ScopeRuntime   Scope = "runtime"
	ScopeLibrary   Scope = "library"
	ScopeFirmware  Scope = "firmware"
	ScopeTransport Scope = "transport"
)

type Outcome string

const (
	OutcomeCompatible     Outcome = "compatible"
	OutcomeIncompatible   Outcome = "incompatible"
	OutcomePatchRequired  Outcome = "patch-required"
	OutcomePatchAvailable Outcome = "patch-available"
	OutcomeSelected       Outcome = "selected"
	OutcomeSkipped        Outcome = "skipped"
)

type FixStrategy string

const (
	FixStrategyPreferCompatible  FixStrategy = "prefer-compatible"
	FixStrategyVersionAwarePatch FixStrategy = "version-aware-patch"
	FixStrategyRuntimeUpdate     FixStrategy = "runtime-update"
	FixStrategyTransportChange   FixStrategy = "transport-change"
)

type Rule struct {
	ID                 string
	Scope              Scope
	Subject            string
	AffectedRange      string
	CompatibleRange    string
	PatchSince         string
	PatchUntil         string
	PatchStrategy      FixStrategy
	PreferredLibrary   string
	PreferredVersion   string
	ReplacementLibrary string
	ReplacementVersion string
	Message            string
	BeginnerMessage    string
	ProfessionalDetail string
	References         []string
}

type Subject struct {
	Scope            Scope                    `json:"scope"`
	Name             string                   `json:"name"`
	Version          string                   `json:"version,omitempty"`
	CoreVersion      string                   `json:"core_version,omitempty"`
	Architecture     string                   `json:"architecture,omitempty"`
	TargetChip       string                   `json:"target_chip,omitempty"`
	TransportKind    string                   `json:"transport_kind,omitempty"`
	AvailableChoices []LibraryCandidateChoice `json:"available_choices,omitempty"`
}

type LibraryCandidateChoice struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Path          string `json:"path,omitempty"`
	Source        string `json:"source,omitempty"`
	Compatibility string `json:"compatibility,omitempty"`
}

type Decision struct {
	Scope              Scope                    `json:"scope"`
	Subject            string                   `json:"subject"`
	RequestedVersion   string                   `json:"requested_version,omitempty"`
	SelectedVersion    string                   `json:"selected_version,omitempty"`
	Outcome            Outcome                  `json:"outcome"`
	Strategy           FixStrategy              `json:"strategy,omitempty"`
	BeginnerMessage    string                   `json:"beginner_message,omitempty"`
	ProfessionalDetail string                   `json:"professional_detail,omitempty"`
	RuleID             string                   `json:"rule_id,omitempty"`
	References         []string                 `json:"references,omitempty"`
	Alternatives       []LibraryCandidateChoice `json:"alternatives,omitempty"`
	Evidence           []string                 `json:"evidence,omitempty"`
	Warnings           []string                 `json:"warnings,omitempty"`
}

type Report struct {
	Subject   Subject            `json:"subject"`
	Status    diagnostics.Status `json:"status"`
	Decisions []Decision         `json:"decisions,omitempty"`
	Warnings  []string           `json:"warnings,omitempty"`
	Errors    []string           `json:"errors,omitempty"`
	Notes     []string           `json:"notes,omitempty"`
}

type CompatibilityReport struct {
	Scope        Scope              `json:"scope"`
	Subject      string             `json:"subject"`
	Status       diagnostics.Status `json:"status"`
	Decisions    []Decision         `json:"decisions,omitempty"`
	Beginner     []string           `json:"beginner_messages,omitempty"`
	Professional []string           `json:"professional_details,omitempty"`
}

type InstallationReport struct {
	PackageName   string                `json:"package_name,omitempty"`
	PackageType   string                `json:"package_type,omitempty"`
	Report        Report                `json:"report"`
	Compatibility []CompatibilityReport `json:"compatibility,omitempty"`
	Stages        []string              `json:"stages,omitempty"`
	Metadata      map[string]string     `json:"metadata,omitempty"`
}

type Resolver struct {
	Rules []Rule
}

func DefaultRules() []Rule {
	return []Rule{
		{
			ID:                 "esp32-espasyncwebserver-mbedtls",
			Scope:              ScopeLibrary,
			Subject:            "ESPAsyncWebServer",
			AffectedRange:      ">=3.3.10",
			CompatibleRange:    ">=3.2.0",
			PatchStrategy:      FixStrategyPreferCompatible,
			PreferredLibrary:   "ESPAsyncWebServer",
			BeginnerMessage:    "use a newer ESP Async WebServer release before patching source",
			ProfessionalDetail: "ESP32 Core 3.3.10 removed old mbedTLS *_ret symbols; prefer a compatible ESP Async WebServer and Async TCP combination instead of patching WebAuthentication.cpp",
			References: []string{
				"mbedtls_md5_starts_ret",
				"mbedtls_md5_update_ret",
				"mbedtls_md5_finish_ret",
			},
		},
	}
}

func DefaultResolver() *Resolver {
	return NewResolver(DefaultRules()...)
}

func NewResolver(rules ...Rule) *Resolver {
	return &Resolver{Rules: append([]Rule(nil), rules...)}
}

func (r *Resolver) Resolve(subject Subject) Report {
	report := Report{
		Subject: subject,
		Status:  diagnostics.StatusPassed,
	}
	for _, rule := range r.rules() {
		if !rule.matches(subject) {
			continue
		}
		decision := rule.decision(subject)
		report.Decisions = append(report.Decisions, decision)
		switch decision.Outcome {
		case OutcomeIncompatible:
			report.Status = diagnostics.StatusFailed
			if decision.BeginnerMessage != "" {
				report.Errors = append(report.Errors, decision.BeginnerMessage)
			}
			if decision.ProfessionalDetail != "" {
				report.Notes = append(report.Notes, decision.ProfessionalDetail)
			}
		case OutcomePatchRequired:
			if report.Status != diagnostics.StatusFailed {
				report.Status = diagnostics.StatusWarning
			}
			if decision.BeginnerMessage != "" {
				report.Warnings = append(report.Warnings, decision.BeginnerMessage)
			}
			if decision.ProfessionalDetail != "" {
				report.Notes = append(report.Notes, decision.ProfessionalDetail)
			}
		case OutcomePatchAvailable:
			if report.Status == diagnostics.StatusPassed {
				report.Status = diagnostics.StatusWarning
			}
			if decision.BeginnerMessage != "" {
				report.Warnings = append(report.Warnings, decision.BeginnerMessage)
			}
		}
	}
	sort.SliceStable(report.Decisions, func(i, j int) bool {
		if report.Decisions[i].Scope != report.Decisions[j].Scope {
			return report.Decisions[i].Scope < report.Decisions[j].Scope
		}
		if report.Decisions[i].Subject != report.Decisions[j].Subject {
			return report.Decisions[i].Subject < report.Decisions[j].Subject
		}
		return report.Decisions[i].Outcome < report.Decisions[j].Outcome
	})
	return report
}

func (r *Resolver) ResolveLibrarySelection(coreVersion, libraryName string, requestedVersion string, candidates []LibraryCandidateChoice) Decision {
	subject := Subject{
		Scope:            ScopeLibrary,
		Name:             libraryName,
		Version:          requestedVersion,
		CoreVersion:      coreVersion,
		AvailableChoices: append([]LibraryCandidateChoice(nil), candidates...),
	}
	report := r.Resolve(subject)
	if len(report.Decisions) == 0 {
		return Decision{
			Scope:              ScopeLibrary,
			Subject:            libraryName,
			RequestedVersion:   requestedVersion,
			SelectedVersion:    requestedVersion,
			Outcome:            OutcomeCompatible,
			BeginnerMessage:    "library selection is compatible",
			ProfessionalDetail: "no compatibility rule matched; requested library version can be used",
		}
	}
	return report.Decisions[0]
}

func (r *Resolver) rules() []Rule {
	if r == nil {
		return nil
	}
	return append([]Rule(nil), r.Rules...)
}

func (r Rule) matches(subject Subject) bool {
	if strings.TrimSpace(r.Subject) != "" && !strings.EqualFold(strings.TrimSpace(r.Subject), strings.TrimSpace(subject.Name)) {
		return false
	}
	switch r.Scope {
	case ScopeLibrary:
		if subject.Scope != ScopeLibrary {
			return false
		}
		if r.AffectedRange == "" {
			return true
		}
		return versionInRange(subject.CoreVersion, r.AffectedRange)
	case ScopeRuntime:
		return subject.Scope == ScopeRuntime && versionInRange(subject.Version, r.AffectedRange)
	case ScopeFirmware:
		return subject.Scope == ScopeFirmware
	case ScopeTransport:
		return subject.Scope == ScopeTransport
	default:
		return false
	}
}

func (r Rule) decision(subject Subject) Decision {
	decision := Decision{
		Scope:              r.Scope,
		Subject:            subject.Name,
		RequestedVersion:   subject.Version,
		SelectedVersion:    subject.Version,
		Outcome:            OutcomeCompatible,
		Strategy:           r.PatchStrategy,
		BeginnerMessage:    r.BeginnerMessage,
		ProfessionalDetail: r.ProfessionalDetail,
		RuleID:             r.ID,
		References:         append([]string(nil), r.References...),
	}
	switch r.PatchStrategy {
	case FixStrategyPreferCompatible:
		if choice, ok := selectCompatibleCandidate(subject.AvailableChoices, r.PreferredLibrary, r.CompatibleRange); ok {
			decision.Outcome = OutcomeSelected
			decision.SelectedVersion = choice.Version
			decision.Alternatives = filterAlternatives(subject.AvailableChoices, choice)
			if decision.BeginnerMessage == "" {
				decision.BeginnerMessage = fmt.Sprintf("selected compatible %s %s", choice.Name, choice.Version)
			}
			if decision.ProfessionalDetail == "" {
				decision.ProfessionalDetail = fmt.Sprintf("preferred compatible candidate %s %s over patching", choice.Name, choice.Version)
			}
		} else if choice, ok := selectPreferredCandidate(subject.AvailableChoices, r.PreferredLibrary, r.PreferredVersion); ok {
			decision.SelectedVersion = choice.Version
			decision.Alternatives = filterAlternatives(subject.AvailableChoices, choice)
			decision.Outcome = OutcomePatchAvailable
			if decision.BeginnerMessage == "" {
				decision.BeginnerMessage = fmt.Sprintf("%s %s is incompatible; a documented patch may be needed", choice.Name, choice.Version)
			}
			if decision.ProfessionalDetail == "" {
				decision.ProfessionalDetail = fmt.Sprintf("no candidate satisfies compatibility range %q; prefer a compatible library version before patching", r.CompatibleRange)
			}
		} else {
			decision.Outcome = OutcomeIncompatible
			if decision.BeginnerMessage == "" {
				decision.BeginnerMessage = "no compatible library version is available"
			}
			if decision.ProfessionalDetail == "" {
				decision.ProfessionalDetail = "no candidate satisfies the compatibility rule and no patch has been documented"
			}
		}
	case FixStrategyVersionAwarePatch:
		decision.Outcome = OutcomePatchAvailable
		if decision.BeginnerMessage == "" {
			decision.BeginnerMessage = "a documented version-aware patch is available"
		}
		if decision.ProfessionalDetail == "" {
			decision.ProfessionalDetail = fmt.Sprintf("apply patch strategy %q for %s", r.ID, subject.Name)
		}
	case FixStrategyRuntimeUpdate:
		decision.Outcome = OutcomePatchRequired
	case FixStrategyTransportChange:
		decision.Outcome = OutcomePatchRequired
	}
	if decision.SelectedVersion == "" {
		decision.SelectedVersion = subject.Version
	}
	return decision
}

func selectPreferredCandidate(candidates []LibraryCandidateChoice, preferredName, preferredVersion string) (LibraryCandidateChoice, bool) {
	if preferredName == "" && preferredVersion == "" {
		return LibraryCandidateChoice{}, false
	}
	for _, candidate := range candidates {
		if preferredName != "" && !strings.EqualFold(candidate.Name, preferredName) {
			continue
		}
		if preferredVersion != "" && candidate.Version != preferredVersion {
			continue
		}
		return candidate, true
	}
	return LibraryCandidateChoice{}, false
}

func selectCompatibleCandidate(candidates []LibraryCandidateChoice, preferredName, compatibleRange string) (LibraryCandidateChoice, bool) {
	var selected LibraryCandidateChoice
	found := false
	for _, candidate := range candidates {
		if preferredName != "" && !strings.EqualFold(candidate.Name, preferredName) {
			continue
		}
		if compatibleRange != "" && !versionInRange(candidate.Version, compatibleRange) {
			continue
		}
		if !found || versionGreater(candidate.Version, selected.Version) {
			selected = candidate
			found = true
		}
	}
	return selected, found
}

func filterAlternatives(candidates []LibraryCandidateChoice, selected LibraryCandidateChoice) []LibraryCandidateChoice {
	var out []LibraryCandidateChoice
	for _, candidate := range candidates {
		if candidate.Name == selected.Name && candidate.Version == selected.Version {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func versionInRange(version, constraint string) bool {
	version = strings.TrimSpace(version)
	constraint = strings.TrimSpace(constraint)
	if version == "" || constraint == "" {
		return false
	}
	parsedVersion, err := semver.Parse(version)
	if err != nil {
		return false
	}
	constraints := strings.Split(constraint, ",")
	for _, c := range constraints {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if strings.HasPrefix(c, ">=") {
			v, err := semver.Parse(strings.TrimSpace(strings.TrimPrefix(c, ">=")))
			if err != nil || parsedVersion.CompareTo(v) < 0 {
				return false
			}
			continue
		}
		if strings.HasPrefix(c, "<=") {
			v, err := semver.Parse(strings.TrimSpace(strings.TrimPrefix(c, "<=")))
			if err != nil || parsedVersion.CompareTo(v) > 0 {
				return false
			}
			continue
		}
		if strings.HasPrefix(c, ">") {
			v, err := semver.Parse(strings.TrimSpace(strings.TrimPrefix(c, ">")))
			if err != nil || parsedVersion.CompareTo(v) <= 0 {
				return false
			}
			continue
		}
		if strings.HasPrefix(c, "<") {
			v, err := semver.Parse(strings.TrimSpace(strings.TrimPrefix(c, "<")))
			if err != nil || parsedVersion.CompareTo(v) >= 0 {
				return false
			}
			continue
		}
		if strings.HasPrefix(c, "=") {
			v, err := semver.Parse(strings.TrimSpace(strings.TrimPrefix(c, "=")))
			if err != nil || parsedVersion.CompareTo(v) != 0 {
				return false
			}
			continue
		}
		v, err := semver.Parse(c)
		if err != nil || parsedVersion.CompareTo(v) != 0 {
			return false
		}
	}
	return true
}

func versionGreater(a, b string) bool {
	if strings.TrimSpace(b) == "" {
		return true
	}
	parsedA, err := semver.Parse(a)
	if err != nil {
		return false
	}
	parsedB, err := semver.Parse(b)
	if err != nil {
		return true
	}
	return parsedA.CompareTo(parsedB) > 0
}

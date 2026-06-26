package compatibility_test

import (
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/compatibility"
	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
	"github.com/stretchr/testify/require"
)

func TestResolveLibrarySelectionPrefersCompatibleVersionOverPatch(t *testing.T) {
	resolver := compatibility.NewResolver(compatibility.Rule{
		ID:                 "espasyncwebserver-mbedtls",
		Scope:              compatibility.ScopeLibrary,
		Subject:            "ESPAsyncWebServer",
		AffectedRange:      ">=3.3.0",
		CompatibleRange:    ">=3.2.0",
		PatchStrategy:      compatibility.FixStrategyPreferCompatible,
		PreferredLibrary:   "ESPAsyncWebServer",
		BeginnerMessage:    "use a newer ESP Async WebServer release for ESP32 Core 3.3.10",
		ProfessionalDetail: "ESPAsyncWebServer 3.1.0 references removed mbedTLS *_ret symbols; select a compatible library release before patching source",
		References:         []string{"mbedtls_md5_starts_ret", "mbedtls_md5_update_ret", "mbedtls_md5_finish_ret"},
	})

	decision := resolver.ResolveLibrarySelection("3.3.10", "ESPAsyncWebServer", "3.1.0", []compatibility.LibraryCandidateChoice{
		{Name: "ESPAsyncWebServer", Version: "3.1.0"},
		{Name: "ESPAsyncWebServer", Version: "3.2.0"},
		{Name: "ESPAsyncWebServer", Version: "3.3.0"},
	})

	require.Equal(t, compatibility.OutcomeSelected, decision.Outcome)
	require.Equal(t, "3.3.0", decision.SelectedVersion)
	require.Contains(t, decision.BeginnerMessage, "newer")
	require.Contains(t, decision.ProfessionalDetail, "mbedTLS")
}

func TestResolveLibrarySelectionFallsBackToDocumentedPatch(t *testing.T) {
	resolver := compatibility.NewResolver(compatibility.Rule{
		ID:                 "espasyncwebserver-mbedtls",
		Scope:              compatibility.ScopeLibrary,
		Subject:            "ESPAsyncWebServer",
		AffectedRange:      ">=3.3.0",
		CompatibleRange:    ">=3.2.0",
		PatchStrategy:      compatibility.FixStrategyVersionAwarePatch,
		BeginnerMessage:    "ESPAsyncWebServer 3.1.0 needs a documented compatibility patch on ESP32 Core 3.3.10",
		ProfessionalDetail: "patch the library only if a documented version-aware fix is required; prefer selecting a compatible library release",
	})

	decision := resolver.ResolveLibrarySelection("3.3.10", "ESPAsyncWebServer", "3.1.0", []compatibility.LibraryCandidateChoice{
		{Name: "ESPAsyncWebServer", Version: "3.1.0"},
	})

	require.Equal(t, compatibility.OutcomePatchAvailable, decision.Outcome)
	require.Equal(t, "3.1.0", decision.SelectedVersion)
	require.Contains(t, decision.BeginnerMessage, "documented")
	require.Contains(t, decision.ProfessionalDetail, "compatible library release")
}

func TestBuildManifestRecordsCompatibilityDecisions(t *testing.T) {
	manifest := firmware.BuildManifest{
		Board:            "esp32s3",
		FQBN:             "esp32:esp32:esp32s3",
		CoreVersion:      "3.3.10",
		ToolchainVersion: "14.2.0",
	}

	manifest.AddCompatibility(compatibility.Decision{
		Scope:              compatibility.ScopeLibrary,
		Subject:            "ESPAsyncWebServer",
		Outcome:            compatibility.OutcomeSelected,
		SelectedVersion:    "3.3.0",
		BeginnerMessage:    "selected compatible library",
		ProfessionalDetail: "compatible library selection recorded in build manifest",
	})

	require.Len(t, manifest.Compatibility, 1)
	require.Equal(t, "ESPAsyncWebServer", manifest.Compatibility[0].Subject)
}

func TestInstallationReportCarriesCompatibilityDecisionSummary(t *testing.T) {
	report := compatibility.InstallationReport{
		PackageName: "esp32",
		PackageType: "platform",
		Report: compatibility.Report{
			Subject: compatibility.Subject{Scope: compatibility.ScopeLibrary, Name: "ESPAsyncWebServer"},
			Status:  diagnostics.StatusWarning,
			Decisions: []compatibility.Decision{{
				Scope:              compatibility.ScopeLibrary,
				Subject:            "ESPAsyncWebServer",
				Outcome:            compatibility.OutcomePatchAvailable,
				BeginnerMessage:    "use a compatible library version first",
				ProfessionalDetail: "documented patch is available if selection cannot be changed",
			}},
			Warnings: []string{"use a compatible library version first"},
			Notes:    []string{"documented patch is available if selection cannot be changed"},
		},
		Compatibility: []compatibility.CompatibilityReport{{
			Scope:     compatibility.ScopeLibrary,
			Subject:   "ESPAsyncWebServer",
			Status:    diagnostics.StatusWarning,
			Decisions: []compatibility.Decision{{Subject: "ESPAsyncWebServer", Outcome: compatibility.OutcomePatchAvailable}},
			Beginner:  []string{"use a compatible library version first"},
		}},
	}

	require.Equal(t, diagnostics.StatusWarning, report.Report.Status)
	require.Len(t, report.Compatibility, 1)
	require.Equal(t, "ESPAsyncWebServer", report.Compatibility[0].Subject)
}

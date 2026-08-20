package chrome

import (
	"reflect"
	"testing"
)

func TestFormatActions(t *testing.T) {
	actions := []DisableAIDownloadAction{
		{Label: "chrome://flags/#foo -> Disabled"},
		{Label: "chrome://flags/#bar -> Default (reset)", Revert: true},
		{
			Label:            GenAIPolicyName + " = 1 (Disabled)",
			Detail:           "some storage location",
			EnterprisePolicy: true,
			PolicyNote:       `Chrome will show the "managed by your organization" banner`,
		},
	}

	got := FormatActions(actions)
	want := []string{
		"  Local flag overrides (chrome://flags):",
		"    - chrome://flags/#foo -> Disabled",
		"  Local flag resets (chrome://flags):",
		"    - chrome://flags/#bar -> Default (reset)",
		"  Enterprise policy (chrome://policy):",
		"    - " + GenAIPolicyName + " = 1 (Disabled)",
		"      some storage location",
		`      ! Chrome will show the "managed by your organization" banner`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FormatActions =\n%v\nwant\n%v", got, want)
	}
}

func TestFormatActions_EmptyGroupsAreOmitted(t *testing.T) {
	// Only a policy action (no flags at all): the two flag section headers
	// must not appear.
	actions := []DisableAIDownloadAction{
		{Label: "policy thing", EnterprisePolicy: true},
	}
	got := FormatActions(actions)
	for _, line := range got {
		if line == "  Local flag overrides (chrome://flags):" || line == "  Local flag resets (chrome://flags):" {
			t.Fatalf("empty section header should be omitted, got lines: %v", got)
		}
	}
}

func TestFormatActions_Nil(t *testing.T) {
	if got := FormatActions(nil); len(got) != 0 {
		t.Fatalf("FormatActions(nil) = %v, want empty", got)
	}
}

func TestFormatSummary(t *testing.T) {
	s := Summary{
		DetectedInstallations: 2,
		PatchedInstallations:  1,
		SkippedInstallations:  1,
		FailedInstallations:   0,
		RestartedExecutables:  1,
	}
	want := "Done. detected=2 patched=1 skipped=1 failed=0 restarted=1"
	if got := FormatSummary(s); got != want {
		t.Fatalf("FormatSummary = %q, want %q", got, want)
	}
}

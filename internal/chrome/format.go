package chrome

import "fmt"

// FormatActions renders actions (as returned by DisableAIDownloadActions)
// into the human-readable preview lines shown before a run: one section
// per group from GroupDisableAIDownloadActions, each action indented under
// its section header, with Detail/PolicyNote as further-indented lines.
// The CLI and GUI previously hand-rolled this same rendering
// independently, with identical section titles/indentation kept in sync by
// hand; this is now the single source of truth for it.
func FormatActions(actions []DisableAIDownloadAction) []string {
	applyFlags, revertFlags, policy := GroupDisableAIDownloadActions(actions)

	var lines []string
	appendSection := func(title string, group []DisableAIDownloadAction) {
		if len(group) == 0 {
			return
		}
		lines = append(lines, "  "+title)
		for _, a := range group {
			lines = append(lines, "    - "+a.Label)
			if a.Detail != "" {
				lines = append(lines, "      "+a.Detail)
			}
			if a.PolicyNote != "" {
				lines = append(lines, "      ! "+a.PolicyNote)
			}
		}
	}

	appendSection("Local flag overrides (chrome://flags):", applyFlags)
	appendSection("Local flag resets (chrome://flags):", revertFlags)
	appendSection("Enterprise policy (chrome://policy):", policy)

	return lines
}

// FormatSummary renders a Summary into the one-line "Done. ..." status
// shown by both the CLI and the GUI after a run.
func FormatSummary(s Summary) string {
	return fmt.Sprintf(
		"Done. detected=%d patched=%d skipped=%d failed=%d restarted=%d",
		s.DetectedInstallations,
		s.PatchedInstallations,
		s.SkippedInstallations,
		s.FailedInstallations,
		s.RestartedExecutables,
	)
}

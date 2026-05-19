package chrome

// GenAIPolicyName is the Chrome Enterprise policy that controls whether the
// Gemini Nano / on-device foundational model is downloaded and used locally.
//
// Value 0 = Allowed (default), 1 = Disabled (do not download, delete cached
// copy). See:
// https://chromeenterprise.google/policies/gen-ai-local-foundational-model-settings/
const GenAIPolicyName = "GenAILocalFoundationalModelSettings"

// PolicyResult is the outcome of applying a host-level Chrome policy.
type PolicyResult struct {
	Applied  bool   // true if the platform store was updated (or would be, in dry-run)
	Location string // human-readable destination (file path, registry key, defaults domain)
	Skipped  string // populated when nothing was written and Applied == false
}

// ApplyDisableAIDownloadPolicy writes GenAILocalFoundationalModelSettings=1
// to the platform's Chrome managed-policy store.
func ApplyDisableAIDownloadPolicy(dryRun bool) (PolicyResult, error) {
	return applyDisableAIDownloadPolicy(dryRun)
}

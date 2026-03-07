cask "go-chrome-ai" do
  arch arm: "arm64", intel: "amd64"

  version "1.0.0"
  sha256 arm: "REPLACE_WITH_DARWIN_ARM64_SHA256",
         intel: "REPLACE_WITH_DARWIN_AMD64_SHA256"

  url "https://github.com/itamaker/go-chrome-ai/releases/download/v#{version}/go-chrome-ai-darwin-#{arch}.tar.gz"
  name "go-chrome-ai"
  desc "Patch Chrome Local State to enable Ask Gemini and other AI features"
  homepage "https://github.com/itamaker/go-chrome-ai"

  binary "go-chrome-ai"

  livecheck do
    url :url
    strategy :github_latest
  end
end

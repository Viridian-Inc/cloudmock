class Cloudmock < Formula
  desc "Local AWS emulation. 25 services. One binary."
  homepage "https://cloudmock.io"
  version "1.0.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/neureaux/cloudmock/releases/download/v1.0.0/cloudmock-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/neureaux/cloudmock/releases/download/v1.0.0/cloudmock-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/neureaux/cloudmock/releases/download/v1.0.0/cloudmock-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/neureaux/cloudmock/releases/download/v1.0.0/cloudmock-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "cloudmock"
    bin.install "cmk"
  end

  test do
    assert_match "CloudMock", shell_output("#{bin}/cloudmock --version")
  end
end

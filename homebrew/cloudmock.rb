class Cloudmock < Formula
  desc "Local AWS emulation. 98 services. One binary."
  homepage "https://cloudmock.io"
  version "1.7.5"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-arm64"
      sha256 "62daf6f3708b3911c58e54dcaa87c826b4f262e7a3f0ff5037f65fde653bf6a4"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-amd64"
      sha256 "77b81c7361ebb22440d5f4784c929388bb6ab47aa1a37cf06adfb27805b1246f"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-arm64"
      sha256 "565745ddcbb1eb68a2d9487bf34ad03b1fec23e64b9c7f12279eec8afec192ad"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-amd64"
      sha256 "940b22b429d0b3ecee5f193fd9805df1d30b0186c1cb79ff83ec5240d262086f"
    end
  end

  def install
    binary = stable.url.split("/").last
    bin.install binary => "cloudmock"
  end

  test do
    assert_match "CloudMock", shell_output("#{bin}/cloudmock --version", 1)
  end
end

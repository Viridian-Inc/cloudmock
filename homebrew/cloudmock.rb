class Cloudmock < Formula
  desc "Local AWS emulation. 98 services. One binary."
  homepage "https://cloudmock.io"
  version "1.10.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-arm64"
      sha256 "c13c1ed2808301ded69b003ef6582d0163912ee246de8d1a769e4512719e21bf"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-amd64"
      sha256 "88c68c67ff4fcc01dea45a32feb4dba9dd3fa49e10a2518c6e4379e61d141980"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-arm64"
      sha256 "cc9a35f61d1bc8ab3361765f6078248cc71f3e8bc0dbc47c808a527758fdffa4"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-amd64"
      sha256 "bc8dcfb25f54a104993ed1d96ab3b2df803f43ab2cd1cc49ff1351b8f2fac33d"
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

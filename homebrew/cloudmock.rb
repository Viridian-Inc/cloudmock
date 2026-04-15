class Cloudmock < Formula
  desc "Local AWS emulation. 98 services. One binary."
  homepage "https://cloudmock.io"
  version "1.8.0"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-arm64"
      sha256 "a88b66b99ed3a1617a56853dc83ed71e586ff0447a658b59bc5fd4318c006796"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-amd64"
      sha256 "d657b97df98b82f85430b0cccb7959079475b0cb87f30b244be2eff85c8217a0"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-arm64"
      sha256 "93159f6188f0170eda9818b76427f7c342a9d1f00318cbad502189b233562652"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-amd64"
      sha256 "286b920186f234dfc104b0ac0added180fcdeed6ed91eee7ce38ace032c93e01"
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

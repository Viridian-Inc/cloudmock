class Cloudmock < Formula
  desc "Local AWS emulation. 98 services. One binary."
  homepage "https://cloudmock.io"
  version "1.8.1"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-arm64"
      sha256 "5838508d10b892c49fe44583a9241649d8fc94938fdf75dded02745837eb6712"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-amd64"
      sha256 "d110d9e85bd324db6858533fa134413c4f1c9b6894425fe140dfce7e4587cc65"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-arm64"
      sha256 "dd4df2a12a17226274c19cc7d769ebfb39333df5af0ec3e99fa15fe3bf6b07c1"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-amd64"
      sha256 "3d526a6b3379a75ac31d3ef58ecaebb166b2d3c894af8931b0e9416ce85dbdc0"
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

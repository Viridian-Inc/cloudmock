class Cloudmock < Formula
  desc "Local AWS emulation. 98 services. One binary."
  homepage "https://cloudmock.io"
  version "1.9.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-arm64"
      sha256 "04e7126cb0adc98437fb485f812592f8c8b2ceb0107df18414d88c6e7ed601ae"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-amd64"
      sha256 "b4b81da957b09db35c57470a740ab20bf934acab0ba2e3af31bf7356b0b8c732"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-arm64"
      sha256 "91af2a93dde2e952397826677c5da0fa474bb592528c568f649165594a480110"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-amd64"
      sha256 "bee979943d54fad91c37f609d0ae8e20c3b728d1d22967aa3e74ad7e7f74c174"
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

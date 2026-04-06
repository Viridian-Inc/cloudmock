class Cloudmock < Formula
  desc "Local AWS emulation. 98 services. One binary."
  homepage "https://cloudmock.io"
  version "1.5.2"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-arm64"
      sha256 "ab661ba05441dfc66ea5b1eef569f4506dd27c457467d74de4e8b74400fd2348"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-darwin-amd64"
      sha256 "2f4e929bd1cda6ac594c7f7b7f4786dbbb5862c5763bf6d44d20266e90de7b98"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-arm64"
      sha256 "3c49359e1d57ca6ce0d9a5a134142de2cd17e2f508a06f0df0ed03d821b249e7"
    end
    on_intel do
      url "https://github.com/Viridian-Inc/cloudmock/releases/download/v#{version}/cloudmock-linux-amd64"
      sha256 "14ab688e13b5936476deacc77bc6438d1a4948c0578374c7a0b0b4b7d9c6e707"
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

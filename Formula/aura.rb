class Aura < Formula
  desc "Hardened, high-aesthetic native terminal ebook reader and knowledge server"
  homepage "https://github.com/Sussysham/aura-go"
  url "https://github.com/Sussysham/aura-go/releases/download/v1.0.0/aura_1.0.0_macOS_amd64.tar.gz"
  sha256 "0000000000000000000000000000000000000000000000000000000000000000"
  version "1.0.0"
  license "MIT"

  def install
    bin.install "aura"
    bin.install "aura-mcp" if File.exist?("aura-mcp")
  end

  test do
    system "#{bin}/aura", "--version"
  end
end

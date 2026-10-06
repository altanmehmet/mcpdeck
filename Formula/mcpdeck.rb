class Mcpdeck < Formula
  desc "Manage MCP servers and global instructions across coding agents"
  homepage "https://github.com/altanmehmet/mcpdeck"
  license "MIT"
  head "https://github.com/altanmehmet/mcpdeck.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", "-trimpath",
           "-ldflags", "-X github.com/altanmehmet/mcpdeck/cmd.version=HEAD",
           "-o", bin/"mcpdeck", "."
  end

  test do
    assert_match "HEAD", shell_output("#{bin}/mcpdeck --version")
    assert_match "Manage MCP servers", shell_output("#{bin}/mcpdeck --help")
  end
end

class Cloudworkstation < Formula
  desc "Launch cloud research environments in seconds"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  license "MIT"
  head "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"

  # This stanza checks the latest release from GitHub
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.2/cws-macos-arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_ARM64" # Will be updated during release process
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.2/cws-macos-amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_DARWIN_AMD64" # Will be updated during release process
    end
  elsif OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.2/cws-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64" # Will be updated during release process
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.2/cws-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64" # Will be updated during release process
    end
  end

  version "0.4.2"

  depends_on "go" => :build

  def install
    # Install binaries from the archive (with platform-specific names)
    platform_suffix = if OS.mac?
                        Hardware::CPU.arm? ? "macos-arm64" : "macos-amd64"
                      else
                        Hardware::CPU.arm? ? "linux-arm64" : "linux-amd64"
                      end
    
    bin.install "cws-#{platform_suffix}" => "cws"
    bin.install "cwsd-#{platform_suffix}" => "cwsd"
    bin.install "cws-gui-#{platform_suffix}" => "cws-gui" if File.exist?("cws-gui-#{platform_suffix}")

    # Install completion scripts
    bash_completion.install "completions/cws.bash" => "cws" if File.exist?("completions/cws.bash")
    zsh_completion.install "completions/cws.zsh" => "_cws" if File.exist?("completions/cws.zsh")
    fish_completion.install "completions/cws.fish" if File.exist?("completions/cws.fish")

    # Install man pages if available
    man1.install "man/cws.1" if File.exist?("man/cws.1")
  end

  def post_install
    # Ensure configuration directory exists
    system "mkdir", "-p", "#{ENV["HOME"]}/.cloudworkstation"
  end

  def caveats
    <<~EOS
      CloudWorkstation #{version} has been installed!
      
      This version includes GUI functionality with system tray integration.
      
      To start the CloudWorkstation daemon:
        cwsd start
        
      To launch your first cloud workstation:
        cws launch python-research my-project
        
      For full documentation:
        cws help
    EOS
  end

  test do
    # Check if binaries can run and report version
    assert_match "CloudWorkstation v#{version}", shell_output("#{bin}/cws --version")
    assert_match "CloudWorkstation Daemon v#{version}", shell_output("#{bin}/cwsd --version")
  end
end
class Cloudworkstation < Formula
  desc "Academic research computing platform - Launch cloud research environments"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  license "MIT"
  head "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"
  
  version "0.4.5"

  # Use prebuilt binaries for faster installation  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.5/cloudworkstation-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER_ARM64_SHA256"
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.5/cloudworkstation-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER_AMD64_SHA256"
    end
  end

  def install
    # Install prebuilt binaries from bin/ directory  
    bin.install "bin/cws"
    bin.install "bin/cwsd"
    
    # Install templates to share/ directory
    share.install "share/templates"
  end

  def post_install
    # Ensure configuration directory exists
    system "mkdir", "-p", "#{ENV["HOME"]}/.cloudworkstation"
  end

  def caveats
    s = <<~EOS
      CloudWorkstation #{version} has been installed with full functionality!
      
      📦 Installed Components:
        • CLI (cws) - Command-line interface with all latest features
        • TUI (cws tui) - Terminal user interface
        • Daemon (cwsd) - Background service
    EOS
    
    if OS.mac?
      s += <<~EOS
        • GUI (cws-gui) - Desktop application with system tray
      EOS
    end
    
    s += <<~EOS
      
      🚀 Quick Start:
        cws profiles add personal research --aws-profile aws --region us-west-2
        cws profiles switch personal
        cws launch "Python Machine Learning (Simplified)" my-project
        
      📚 Documentation:
        cws help                    # Full command reference (Cobra CLI)
        cws templates               # List available templates
        cws daemon status           # Check daemon status
        
      🔧 Service Management (Auto-Start on Boot):
        brew services start cloudworkstation   # Auto-start daemon with Homebrew
        brew services stop cloudworkstation    # Stop daemon service
        brew services restart cloudworkstation # Restart daemon service
      
      🛡️ Version 0.4.4 Security Update:
        Web interfaces (Jupyter, RStudio) now require SSH port forwarding for security.
        Example: ssh -L 8888:localhost:8888 user@instance
        
        This prevents internet exposure while maintaining full functionality.
        
      Note: Version 0.4.4 includes enhanced security and prebuilt binaries for fast installation.
    EOS
  end

  # Homebrew automatically handles service management during install/uninstall

  test do
    # Test that binaries exist and are executable
    assert_predicate bin/"cws", :exist?
    assert_predicate bin/"cwsd", :exist?
    
    # Test version command
    assert_match "CloudWorkstation CLI v", shell_output("#{bin}/cws --version")
    assert_match "CloudWorkstation Daemon v", shell_output("#{bin}/cwsd --version")
  end

  service do
    run [opt_bin/"cwsd"]
    keep_alive true
    log_path var/"log/cloudworkstation/cwsd.log"
    error_log_path var/"log/cloudworkstation/cwsd.log"
    working_dir HOMEBREW_PREFIX
  end
end
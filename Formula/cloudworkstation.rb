class Cloudworkstation < Formula
  desc "Academic research computing platform - Launch cloud research environments"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  license "MIT"
  head "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"
  
  version "0.4.4"

  # Use prebuilt binaries for faster installation  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.4/cloudworkstation-darwin-arm64.tar.gz"
      sha256 "b704632a37db2663a425d6388d1c17d08e7bfec867a5ee2467225d093800d873"
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.4/cloudworkstation-darwin-amd64.tar.gz"
      sha256 "217fba92718bb02617bd16e90940709e918b18922b92130da49efd42c299956c"
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
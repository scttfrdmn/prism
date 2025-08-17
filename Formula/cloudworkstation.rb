class Cloudworkstation < Formula
  desc "Enterprise research management platform - Launch cloud research environments in seconds"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  license "MIT"
  head "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"
  
  version "0.4.2"

  # Use prebuilt binaries for faster installation  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.2/cloudworkstation-darwin-arm64.tar.gz"
      sha256 "831792e74d5d80325e14d3ad0d73600074958170d5deadf3159e332a6cd789f7"
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.2/cloudworkstation-darwin-amd64.tar.gz"
      sha256 "ef8be5312ba9c4b9848b6e223a7fead762449249f5db0315e888adbb2d1685ba"
    end
  end

  def install
    # Install prebuilt binaries directly from working directory
    bin.install "cws"
    bin.install "cwsd"
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
        
      🔧 Service Management:
        brew services start cloudworkstation   # Auto-start daemon
        brew services stop cloudworkstation    # Stop daemon service
      
      Note: Version 0.4.2 includes enterprise research features with prebuilt binaries for fast installation.
    EOS
  end

  test do
    # Test that binaries exist and are executable
    assert_predicate bin/"cws", :exist?
    assert_predicate bin/"cwsd", :exist?
    
    # Test version command
    assert_match "CloudWorkstation v", shell_output("#{bin}/cws --version")
    assert_match "CloudWorkstation v", shell_output("#{bin}/cwsd --version")
  end

  service do
    run [opt_bin/"cwsd"]
    keep_alive true
    log_path var/"log/cloudworkstation/cwsd.log"
    error_log_path var/"log/cloudworkstation/cwsd.log"
    working_dir HOMEBREW_PREFIX
  end
end
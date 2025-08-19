class Cloudworkstation < Formula
  desc "Academic research computing platform - Launch cloud research environments"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  license "MIT"
  head "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"
  
  version "0.4.3"

  # Use prebuilt binaries for faster installation  
  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.3/cloudworkstation-darwin-arm64.tar.gz"
      sha256 "1bf41b76441f14c6cd05a51f6081d5e68f6d16609e1ac680f390aefc9be4d28b"
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v0.4.3/cloudworkstation-darwin-amd64.tar.gz"
      sha256 "8fee1c699c715d81771f1c2ff82958d8af99debacccf7d2e038813e91fe83a8a"
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
        
      🔧 Service Management (Auto-Start on Boot):
        brew services start cloudworkstation   # Auto-start daemon with Homebrew
        brew services stop cloudworkstation    # Stop daemon service
        brew services restart cloudworkstation # Restart daemon service
      
      Note: Version 0.4.3 includes research computing features with prebuilt binaries for fast installation.
    EOS
  end

  def uninstall
    # Stop Homebrew service if running
    quiet_system("brew", "services", "stop", "cloudworkstation") if which("brew")
    
    # Attempt graceful daemon shutdown via API
    if File.exist?("#{bin}/cws")
      puts "🛑 Attempting graceful daemon shutdown..."
      system("#{bin}/cws", "daemon", "stop")
      sleep 2
    end
    
    # Find and terminate any remaining daemon processes
    puts "🔍 Checking for remaining daemon processes..."
    daemon_pids = `pgrep -f cwsd 2>/dev/null || true`.strip.split("\n")
    
    unless daemon_pids.empty?
      puts "⚠️  Found #{daemon_pids.length} daemon processes, terminating..."
      daemon_pids.each do |pid|
        next if pid.strip.empty?
        puts "  Stopping PID #{pid}"
        # Try graceful termination first
        system("kill", "-TERM", pid.strip)
      end
      
      sleep 3
      
      # Force kill any remaining processes
      remaining_pids = `pgrep -f cwsd 2>/dev/null || true`.strip.split("\n")
      unless remaining_pids.empty?
        puts "🔨 Force killing remaining processes..."
        remaining_pids.each do |pid|
          next if pid.strip.empty?
          puts "  Force killing PID #{pid}"
          system("kill", "-KILL", pid.strip)
        end
      end
    end
    
    # Clean up configuration and data files
    puts "🧹 Cleaning up CloudWorkstation files..."
    
    config_dir = "#{ENV['HOME']}/.cloudworkstation"
    if Dir.exist?(config_dir)
      puts "  Removing config directory: #{config_dir}"
      rm_rf(config_dir)
    end
    
    # Clean up log files
    log_dir = "#{ENV['HOME']}/Library/Logs/cloudworkstation"
    if Dir.exist?(log_dir)
      puts "  Removing log directory: #{log_dir}"
      rm_rf(log_dir)
    end
    
    # Remove Homebrew service files
    service_file = "#{ENV['HOME']}/Library/LaunchAgents/homebrew.mxcl.cloudworkstation.plist"
    if File.exist?(service_file)
      puts "  Removing service file: #{service_file}"
      rm_f(service_file)
    end
    
    # Clean up any remaining state files
    [
      "#{ENV['HOME']}/.cloudworkstation",
      "/tmp/cloudworkstation*",
      "/var/tmp/cloudworkstation*"
    ].each do |pattern|
      Dir.glob(pattern).each do |path|
        puts "  Removing: #{path}"
        rm_rf(path) if File.exist?(path)
      end
    end
    
    puts "✅ CloudWorkstation uninstallation completed"
    puts ""
    puts "Note: AWS credentials and profiles remain unchanged."
    puts "If you want to remove AWS credentials, run: aws configure"
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
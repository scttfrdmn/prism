class Cloudworkstation < Formula
  desc "CLI tool for launching pre-configured cloud workstations for academic research"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  url "https://github.com/scttfrdmn/cloudworkstation/archive/v0.4.5.tar.gz"
  sha256 "PLACEHOLDER_SOURCE_SHA256"
  license "MIT"
  version "0.4.5"

  depends_on "go" => :build

  def install
    # Ensure dependencies are up to date
    system "go", "mod", "tidy"
    
    # Build all binaries
    system "make", "build"
    
    # Install binaries
    bin.install "bin/cws"
    bin.install "bin/cwsd"
    
    # Install GUI if available (optional)
    if File.exist?("bin/cws-gui")
      bin.install "bin/cws-gui"
    end
    
    # Install documentation
    doc.install "README.md"
    doc.install "CLAUDE.md"
    doc.install "CHANGELOG.md"
    
    # Install templates
    share.install "templates"
    
    # Install man pages if they exist
    man1.install Dir["docs/man/*.1"] if Dir.exist?("docs/man")
  end

  def caveats
    gui_available = (bin/"cws-gui").exist?
    
    caveat_text = <<~EOS
      CloudWorkstation has been installed with multiple interfaces:
      
      Command Line Interface (CLI):
        cws --help
      
      Terminal User Interface (TUI):
        cws tui
    EOS
    
    if gui_available
      caveat_text += <<~EOS
      
      Graphical User Interface (GUI):
        cws-gui
        
      GUI Startup Options:
        cws-gui -autostart        # Configure auto-start at login
        cws-gui -remove-autostart # Remove auto-start
        cws-gui -help            # Show GUI help
      EOS
    else
      caveat_text += <<~EOS
      
      Note: GUI not available (requires Wails CLI for building)
        Install GUI support: go install github.com/wailsapp/wails/v3/cmd/wails@latest
        Then reinstall: brew reinstall cloudworkstation
      EOS
    end
    
    caveat_text += <<~EOS
      
      To get started:
        cws templates
        cws launch <template-name> <instance-name>
      
      The daemon (cwsd) will start automatically when needed.
      You can also start it manually or as a service.
      
      AWS credentials are required. Set them up with:
        aws configure
      
      For more information:
        https://github.com/scttfrdmn/cloudworkstation
    EOS
    
    caveat_text
  end

  test do
    # Test that binaries exist and are executable
    assert_predicate bin/"cws", :exist?
    assert_predicate bin/"cwsd", :exist?
    
    # Test GUI if available (optional)
    if (bin/"cws-gui").exist?
      assert_predicate bin/"cws-gui", :exist?
    end
    
    # Test version command
    output = shell_output("#{bin}/cws version 2>&1", 0)
    assert_match "CloudWorkstation v#{version}", output
    
    # Test templates command (should work without AWS credentials)
    system "#{bin}/cws", "templates"
  end

  service do
    run [opt_bin/"cwsd"]
    keep_alive true
    log_path var/"log/cloudworkstation/cwsd.log"
    error_log_path var/"log/cloudworkstation/cwsd.log"
    working_dir HOMEBREW_PREFIX
  end
end
class Cloudworkstation < Formula
  desc "CLI tool for launching pre-configured cloud workstations for academic research"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  url "https://github.com/scttfrdmn/cloudworkstation/archive/v0.4.1.tar.gz"
  sha256 "37e65b29bff6cc4d69b4cee238856dc8dfa993f3196a27d5cabe4cb2c1726145"
  license "MIT"
  version "0.4.1"

  depends_on "go" => :build

  def install
    # Ensure dependencies are up to date
    system "go", "mod", "tidy"
    
    # Build all binaries
    system "make", "build"
    
    # Install binaries
    bin.install "bin/cws"
    bin.install "bin/cwsd" 
    bin.install "bin/cws-gui"
    
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
    <<~EOS
      CloudWorkstation has been installed with three interfaces:
      
      Command Line Interface (CLI):
        cws --help
      
      Terminal User Interface (TUI):
        cws tui
      
      Graphical User Interface (GUI):
        cws-gui
      
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
  end

  test do
    # Test that binaries exist and are executable
    assert_predicate bin/"cws", :exist?
    assert_predicate bin/"cwsd", :exist?  
    assert_predicate bin/"cws-gui", :exist?
    
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
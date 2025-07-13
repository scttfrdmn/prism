class Cloudworkstation < Formula
  desc "Launch pre-configured research environments in the cloud in seconds"
  homepage "https://github.com/scttfrdmn/cloudworkstation"
  version "0.4.1"
  license "MIT"

  # Testing tap configuration
  head do
    url "https://github.com/scttfrdmn/cloudworkstation.git", branch: "main"
  end

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v#{version}/cws-macos-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILDING"
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v#{version}/cws-macos-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILDING"
    end
  elsif OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v#{version}/cws-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILDING"
    else
      url "https://github.com/scttfrdmn/cloudworkstation/releases/download/v#{version}/cws-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_BUILDING"
    end
  end

  depends_on "go" => :build

  def install
    # Binary is pre-built in the tarball, just copy it to bin
    bin.install "cws"
    bin.install "cwsd"
    
    # Install bash completion
    output = Utils.safe_popen_read(bin/"cws", "completion", "bash")
    (bash_completion/"cws").write output
    
    # Install zsh completion
    output = Utils.safe_popen_read(bin/"cws", "completion", "zsh")
    (zsh_completion/"_cws").write output
  end

  def caveats
    <<~EOS
      CloudWorkstation requires AWS credentials to function.
      
      If you haven't set up AWS credentials yet, run:
        aws configure
      
      Or set up credentials directly with:
        cws config profile your_profile_name
        cws config region your_aws_region
    EOS
  end

  test do
    assert_match "CloudWorkstation v#{version}", shell_output("#{bin}/cws version")
  end
end
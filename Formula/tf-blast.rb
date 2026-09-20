# typed: false
# frozen_string_literal: true

# This formula was auto-generated for tf-blast (https://github.com/smford/tf-blast).
class TfBlast < Formula
  desc "Ultra-fast, zero-trust blast radius analyzer for Terraform and OpenTofu"
  homepage "https://github.com/smford/tf-blast"
  version "1.0.0"
  license "AGPL-3.0-or-later"

  on_macos do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_arm64.tar.gz"
      sha256 "1326ca2d0e27fedc6da8f6d8f0778b9566b7170d5c4553dc7d517b292628bb4d"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_amd64.tar.gz"
      sha256 "169e842386537bbfa12309c91ce06b0ff5cf7fb31612cb64f6c74119a5c9de30"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_arm64.tar.gz"
      sha256 "27fa3de99013d596f5ce76ceb3c28848ac06515abaa1566ad6e3ee427d08847f"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_amd64.tar.gz"
      sha256 "6efbce4531a7b8746c70b307d2fef3fbbd6285d0063495b004eb34a9917f9697"
    end
  end

  def install
    bin.install "tf-blast"
    generate_completions_from_executable(bin/"tf-blast", "completion")
  end

  test do
    assert_match "tf-blast version", shell_output("#{bin}/tf-blast --version")
  end
end

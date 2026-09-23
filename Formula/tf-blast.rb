# typed: false
# frozen_string_literal: true

# This formula was auto-generated for tf-blast (https://github.com/smford/tf-blast).
class TfBlast < Formula
  desc "Ultra-fast, zero-trust blast radius analyzer for Terraform and OpenTofu"
  homepage "https://github.com/smford/tf-blast"
  version "1.2.2"
  license "AGPL-3.0-or-later"

  on_macos do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_arm64.tar.gz"
      sha256 "330a3828735a6d14c565916cf8d608f151614a854eaf0a7122505722cb7ef0e8"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_amd64.tar.gz"
      sha256 "0ff1ad6c2419e0555b1d5ae83145470fda8c0335053aa7235ee20ff88d4b39f2"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_arm64.tar.gz"
      sha256 "b7259f63fdae976a75dee53860a869497d92ed9f4deec67e834885da93735a1a"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_amd64.tar.gz"
      sha256 "7c95d9711375b9332dc792d6f3b7f59c2a7447f21ffcc6daa1e1812c354b9ad0"
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

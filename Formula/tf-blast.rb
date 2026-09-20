# typed: false
# frozen_string_literal: true

# This formula was auto-generated for tf-blast (https://github.com/smford/tf-blast).
class TfBlast < Formula
  desc "Ultra-fast, zero-trust blast radius analyzer for Terraform and OpenTofu"
  homepage "https://github.com/smford/tf-blast"
  version "1.2.1"
  license "AGPL-3.0-or-later"

  on_macos do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_arm64.tar.gz"
      sha256 "22a34993a44148c81455fa34f8aeff44e43424ac5f53356e85f9096626ef3fe0"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_amd64.tar.gz"
      sha256 "04d346fed756bf10aa3a4eb55ffa5f2074e368898403efc452d55b0bb0d6914d"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_arm64.tar.gz"
      sha256 "22839550259ae50cac8cd31a26bc77aed633f514e27cc879ad52b0b8a32cccd3"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_amd64.tar.gz"
      sha256 "da243d621aa4e489d5c845c4b9ec214e20d9b89c9919afe584eeaf8bf0fef0eb"
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

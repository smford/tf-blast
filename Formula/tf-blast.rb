# typed: false
# frozen_string_literal: true

# This formula was auto-generated for tf-blast (https://github.com/smford/tf-blast).
class TfBlast < Formula
  desc "Ultra-fast, zero-trust blast radius analyzer for Terraform and OpenTofu"
  homepage "https://github.com/smford/tf-blast"
  version "1.1.1"
  license "AGPL-3.0-or-later"

  on_macos do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_arm64.tar.gz"
      sha256 "38ca1631a273e16f65111343247a3d92be4adbb930ca7d38224bd8136fdb978e"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_darwin_amd64.tar.gz"
      sha256 "f469ea721e8d0e4300ce132a4a956e99c2e2067cad40d5ec7f7a7884f40dcee9"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_arm64.tar.gz"
      sha256 "46feee25098cc47fe88c551a87e1ffa48e3b456f5e6c0f3d224e52e9fa08124a"
    end
    on_intel do
      url "https://github.com/smford/tf-blast/releases/download/v#{version}/tf-blast_#{version}_linux_amd64.tar.gz"
      sha256 "1f331af5cd45866d557915a4ec1294686cb83a3a1af13ecafca3993ee9cdfd30"
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

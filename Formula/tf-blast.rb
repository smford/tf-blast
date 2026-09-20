class TfBlast < Formula
  desc "Ultra-fast, zero-trust blast radius analyzer for Terraform and OpenTofu"
  homepage "https://github.com/smford/tf-blast"
  url "https://github.com/smford/tf-blast/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "Apache-2.0"
  head "https://github.com/smford/tf-blast.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X main.Version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/tf-blast"

    generate_completions_from_executable(bin/"tf-blast", "completion")
  end

  test do
    assert_match "tf-blast version", shell_output("#{bin}/tf-blast --version")
  end
end

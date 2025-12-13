class Sshthing < Formula
  desc "Secure SSH manager TUI (SQLCipher + AES-GCM) with SSH/SFTP and Finder mounts"
  homepage "https://github.com/Vansh-Raja/SSHThing"

  # For an initial tap-based install, ship a HEAD-only formula.
  # Once you cut releases, replace this with `url` + `sha256` blocks.
  head "https://github.com/Vansh-Raja/SSHThing.git", branch: "homebrew-tap"

  depends_on "go" => :build
  depends_on "sqlcipher"

  def install
    # go-sqlcipher uses CGO and links against SQLCipher.
    ENV["CGO_ENABLED"] = "1"
    ENV.append "CGO_CPPFLAGS", "-I#{Formula["sqlcipher"].opt_include}"
    ENV.append "CGO_LDFLAGS", "-L#{Formula["sqlcipher"].opt_lib}"

    ldflags = %W[
      -s -w
      -X main.version=#{version}
    ]

    system "go", "build", *std_go_args(output: bin/"sshthing", ldflags: ldflags), "./cmd/sshthing"
  end

  test do
    assert_match "sshthing", shell_output("#{bin}/sshthing --version")
  end
end

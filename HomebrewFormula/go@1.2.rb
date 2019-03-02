class GoAT12 < Formula
  desc "Go programming environment (1.2)"
  homepage "https://golang.org"
  url "https://dl.google.com/go/go1.2.2.src.tar.gz"
  # sha1 "3ce0ac4db434fc1546fec074841ff40dc48c1167"
  sha256 "fbcfe1fe6dfe660cae1c973811c5e2075e3f7b06feea32b4b91c7f0b48352391"
  version "1.2.2"

  keg_only :versioned_formula

  def install
    ENV.refurbish_args

    cd "src" do
      ENV["GOROOT_FINAL"] = libexec
      ENV["CGO_ENABLED"]  = "0"
      system "./make.bash", "--no-clean"
    end

    (buildpath/"pkg/obj").rmtree
    libexec.install Dir["*"]
    (bin/"go").write_env_script(libexec/"bin/go", :PATH => "#{libexec}/bin:$PATH")
    bin.install_symlink libexec/"bin/gofmt"
  end

  def caveats; <<~EOS
    As of go 1.2, a valid GOPATH is required to use the `go get` command:
      https://golang.org/doc/code.html#GOPATH

    You may wish to add the GOROOT-based install location to your PATH:
      export PATH=$PATH:#{opt_libexec}/bin
    EOS
  end

  test do
    (testpath/"hello.go").write <<~EOS
      package main
      import "fmt"
      func main() {
          fmt.Println("Hello World")
      }
    EOS
    # Run go fmt check for no errors then run the program.
    # This is a a bare minimum of go working as it uses fmt, build, and run.
    system "#{bin}/go", "fmt", "hello.go"
    assert_equal "Hello World\n", shell_output("#{bin}/go run hello.go")
  end
end

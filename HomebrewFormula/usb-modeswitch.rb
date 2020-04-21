class UsbModeswitch < Formula
  desc "Handling Mode-Switching USB Devices"
  homepage "http://www.draisberghof.de/usb_modeswitch/"
  url "http://www.draisberghof.de/usb_modeswitch/usb-modeswitch-2.5.2.tar.bz2"
  sha256 "abffac09c87eacd78e101545967dc25af7e989745b4276756d45dbf4008a2ea6"

  depends_on "pkg-config" => :build
  depends_on "libusb-compat"

  resource "usb-modeswitch-data-20170806.tar.bz2" do
    url "http://www.draisberghof.de/usb_modeswitch/usb-modeswitch-data-20170806.tar.bz2"
    sha256 "ce413ef2a50e648e9c81bc3ea6110e7324a8bf981034fc9ec4467d3562563c2c"
  end

  def install
    system "make", "PREFIX=#{prefix}"
    bin.install "usb_modeswitch"

    resource("usb-modeswitch-data-20170806.tar.bz2").stage do
      (share/"usb_modeswitch").install Dir["usb_modeswitch.d/*"]
    end
  end

  test do
    system "#{bin}/usb_modeswitch", "--help"
  end
end

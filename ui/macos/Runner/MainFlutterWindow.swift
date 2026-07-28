import Cocoa
import FlutterMacOS

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    self.titlebarAppearsTransparent = true
    self.titleVisibility = .hidden
    self.styleMask.insert(.fullSizeContentView)
    self.backgroundColor = NSColor.black
    self.minSize = NSSize(width: 890, height: 600)

    let channel = FlutterMethodChannel(
      name: "vdub/window",
      binaryMessenger: flutterViewController.engine.binaryMessenger)
    channel.setMethodCallHandler { [weak self] (call, result) in
      if call.method == "setDarkMode" {
        let isDark = call.arguments as? Bool ?? true
        self?.appearance = NSAppearance(named: isDark ? .darkAqua : .aqua)
        self?.backgroundColor = isDark
          ? NSColor.black
          : NSColor(red: 0.95, green: 0.95, blue: 0.97, alpha: 1)
        result(nil)
      } else {
        result(FlutterMethodNotImplemented)
      }
    }

    RegisterGeneratedPlugins(registry: flutterViewController)

    super.awakeFromNib()
  }
}

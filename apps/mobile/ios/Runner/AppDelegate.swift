import Flutter
import UIKit

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  private var mobileIdentityChannel: FlutterMethodChannel?
  private let mobileIdentityStorage = MobileIdentityStorage()

  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    let channel = FlutterMethodChannel(
      name: "dev.coderoam/mobile_identity_storage_v1",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    channel.setMethodCallHandler { [mobileIdentityStorage] call, result in
      do {
        switch call.method {
        case "read":
          result(try mobileIdentityStorage.read())
        case "createIfAbsent":
          guard let arguments = call.arguments as? [String: Any],
                let value = arguments["value"] as? String
          else {
            throw MobileIdentityStorageError.unavailable
          }
          result(try mobileIdentityStorage.createIfAbsent(value))
        default:
          result(FlutterMethodNotImplemented)
        }
      } catch {
        result(
          FlutterError(
            code: "identity_storage_unavailable",
            message: "Identity storage is unavailable.",
            details: nil
          )
        )
      }
    }
    mobileIdentityChannel = channel
  }
}

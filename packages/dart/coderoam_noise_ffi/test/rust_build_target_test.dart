import 'package:code_assets/code_assets.dart';
import 'package:test/test.dart';

import '../hook/rust_build_target.dart';

void main() {
  test('maps iOS device and simulator targets with the deployment floor', () {
    final device = resolveRustBuildTarget(
      operatingSystem: OS.iOS,
      architecture: Architecture.arm64,
      iOSSDK: IOSSdk.iPhoneOS,
      iOSVersion: 13,
    );
    final simulator = resolveRustBuildTarget(
      operatingSystem: OS.iOS,
      architecture: Architecture.x64,
      iOSSDK: IOSSdk.iPhoneSimulator,
      iOSVersion: 13,
    );

    expect(device.triple, 'aarch64-apple-ios');
    expect(device.environment, {'IPHONEOS_DEPLOYMENT_TARGET': '13.0'});
    expect(simulator.triple, 'x86_64-apple-ios');
  });

  test('uses the NDK API-specific clang wrapper for Android', () {
    final target = resolveRustBuildTarget(
      operatingSystem: OS.android,
      architecture: Architecture.arm64,
      androidAPI: 21,
      compiler: Uri.file('/ndk/toolchains/llvm/prebuilt/host/bin/clang'),
    );

    expect(target.triple, 'aarch64-linux-android');
    expect(target.environment, {
      'CARGO_TARGET_AARCH64_LINUX_ANDROID_LINKER':
          '/ndk/toolchains/llvm/prebuilt/host/bin/'
          'aarch64-linux-android21-clang',
    });
  });

  test('rejects an unsupported iOS device architecture', () {
    expect(
      () => resolveRustBuildTarget(
        operatingSystem: OS.iOS,
        architecture: Architecture.x64,
        iOSSDK: IOSSdk.iPhoneOS,
        iOSVersion: 13,
      ),
      throwsUnsupportedError,
    );
  });
}

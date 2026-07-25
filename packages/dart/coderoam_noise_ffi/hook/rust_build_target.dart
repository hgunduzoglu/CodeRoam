import 'package:code_assets/code_assets.dart';

final class RustBuildTarget {
  const RustBuildTarget({required this.triple, this.environment = const {}});

  final String triple;
  final Map<String, String> environment;
}

RustBuildTarget resolveRustBuildTarget({
  required OS operatingSystem,
  required Architecture architecture,
  IOSSdk? iOSSDK,
  int? iOSVersion,
  int? androidAPI,
  Uri? compiler,
}) {
  if (operatingSystem == OS.iOS) {
    if (iOSSDK == null || iOSVersion == null) {
      throw UnsupportedError(
        'The iOS Rust target configuration is incomplete.',
      );
    }
    final triple = switch ((architecture, iOSSDK)) {
      (Architecture.arm64, IOSSdk.iPhoneOS) => 'aarch64-apple-ios',
      (Architecture.arm64, IOSSdk.iPhoneSimulator) => 'aarch64-apple-ios-sim',
      (Architecture.x64, IOSSdk.iPhoneSimulator) => 'x86_64-apple-ios',
      _ => throw UnsupportedError(
        'The iOS Rust target does not support $architecture/$iOSSDK.',
      ),
    };
    return RustBuildTarget(
      triple: triple,
      environment: {'IPHONEOS_DEPLOYMENT_TARGET': '$iOSVersion.0'},
    );
  }

  if (operatingSystem == OS.android) {
    if (androidAPI == null || compiler == null) {
      throw UnsupportedError(
        'The Android Rust target configuration is incomplete.',
      );
    }
    final (triple, clangTarget) = switch (architecture) {
      Architecture.arm => (
        'armv7-linux-androideabi',
        'armv7a-linux-androideabi',
      ),
      Architecture.arm64 => ('aarch64-linux-android', 'aarch64-linux-android'),
      Architecture.x64 => ('x86_64-linux-android', 'x86_64-linux-android'),
      _ => throw UnsupportedError(
        'The Android Rust target does not support $architecture.',
      ),
    };
    final linker = compiler.resolve('$clangTarget$androidAPI-clang');
    final linkerKey =
        'CARGO_TARGET_${triple.toUpperCase().replaceAll('-', '_')}_LINKER';
    return RustBuildTarget(
      triple: triple,
      environment: {linkerKey: linker.toFilePath()},
    );
  }

  final triple = switch ((operatingSystem, architecture)) {
    (OS.macOS, Architecture.arm64) => 'aarch64-apple-darwin',
    (OS.macOS, Architecture.x64) => 'x86_64-apple-darwin',
    (OS.linux, Architecture.arm64) => 'aarch64-unknown-linux-gnu',
    (OS.linux, Architecture.x64) => 'x86_64-unknown-linux-gnu',
    _ => throw UnsupportedError(
      'The Rust target does not support $operatingSystem/$architecture.',
    ),
  };
  return RustBuildTarget(triple: triple);
}

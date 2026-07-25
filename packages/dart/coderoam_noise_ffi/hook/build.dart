import 'dart:io';

import 'package:code_assets/code_assets.dart';
import 'package:hooks/hooks.dart';

import 'cargo_process.dart';

void main(List<String> arguments) async {
  await build(arguments, (input, output) async {
    if (!input.config.buildCodeAssets) {
      return;
    }

    final code = input.config.code;
    final rustTarget = switch ((code.targetOS, code.targetArchitecture)) {
      (OS.macOS, Architecture.arm64) => 'aarch64-apple-darwin',
      (OS.macOS, Architecture.x64) => 'x86_64-apple-darwin',
      (OS.linux, Architecture.arm64) => 'aarch64-unknown-linux-gnu',
      (OS.linux, Architecture.x64) => 'x86_64-unknown-linux-gnu',
      _ => throw BuildError(
        message:
            'The current CodeRoam Noise FFI probe supports macOS and Linux '
            'hosts only, not ${code.targetOS}/${code.targetArchitecture}.',
      ),
    };
    final manifest = input.packageRoot.resolve('rust/Cargo.toml');
    final lockfile = input.packageRoot.resolve('rust/Cargo.lock');
    final source = input.packageRoot.resolve('rust/src/lib.rs');
    final cargoProcessSource = input.packageRoot.resolve(
      'hook/cargo_process.dart',
    );
    final cargoRunner = input.packageRoot.resolve('hook/cargo_runner.dart');
    final cargoTargetDirectory = input.outputDirectory.resolve('cargo/');

    output.dependencies.addAll([
      manifest,
      lockfile,
      source,
      cargoProcessSource,
      cargoRunner,
    ]);

    late final int exitCode;
    try {
      exitCode = await runCargoProcess(
        executable: 'cargo',
        arguments: [
          'build',
          '--locked',
          '--release',
          '--manifest-path',
          manifest.toFilePath(),
          '--target',
          rustTarget,
        ],
        workingDirectory: input.packageRoot.toFilePath(),
        environment: {
          ...Platform.environment,
          'CARGO_TARGET_DIR': cargoTargetDirectory.toFilePath(),
          'CARGO_TERM_COLOR': 'never',
        },
        runnerScript: cargoRunner,
        scratchDirectory: input.outputDirectory.resolve('cargo-process/'),
      );
    } on CargoProcessTimeout {
      throw BuildError(message: 'Cargo timed out while building Noise FFI.');
    } on CargoProcessCleanupFailure {
      throw BuildError(
        message: 'Cargo did not stop cleanly while building Noise FFI.',
      );
    }
    if (exitCode != 0) {
      throw BuildError(message: 'Cargo failed to build Noise FFI.');
    }

    final library = cargoTargetDirectory.resolve(
      '$rustTarget/release/'
      '${code.targetOS.dylibFileName('coderoam_noise_ffi')}',
    );
    if (!await File.fromUri(library).exists()) {
      throw BuildError(message: 'Cargo did not produce the Noise FFI library.');
    }

    output.assets.code.add(
      CodeAsset(
        package: input.packageName,
        name: 'coderoam_noise_ffi.dart',
        linkMode: DynamicLoadingBundled(),
        file: library,
      ),
    );
  });
}

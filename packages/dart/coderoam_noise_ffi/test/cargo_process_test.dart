import 'dart:io';

import 'package:test/test.dart';

import '../hook/cargo_process.dart';

void main() {
  test('timeout terminates Cargo and its persistent child process', () async {
    final packageRoot = Directory.current.uri;
    final temporaryDirectory = await Directory.systemTemp.createTemp(
      'coderoam-cargo-process-',
    );
    addTearDown(() => temporaryDirectory.delete(recursive: true));
    final parentPidFile = File.fromUri(
      temporaryDirectory.uri.resolve('parent.pid'),
    );
    final childPidFile = File.fromUri(
      temporaryDirectory.uri.resolve('child.pid'),
    );

    await expectLater(
      runCargoProcess(
        executable: Platform.resolvedExecutable,
        arguments: [
          packageRoot.resolve('test/fixtures/fake_cargo.dart').toFilePath(),
          packageRoot
              .resolve('test/fixtures/persistent_child.dart')
              .toFilePath(),
          parentPidFile.path,
          childPidFile.path,
        ],
        workingDirectory: temporaryDirectory.path,
        environment: Platform.environment,
        runnerScript: packageRoot.resolve('hook/cargo_runner.dart'),
        scratchDirectory: temporaryDirectory.uri.resolve('runner/'),
        timeout: const Duration(seconds: 3),
        terminationGrace: const Duration(milliseconds: 250),
        killGrace: const Duration(seconds: 2),
      ),
      throwsA(isA<CargoProcessTimeout>()),
    );

    final parentPid = int.parse(await parentPidFile.readAsString());
    final childPid = int.parse(await childPidFile.readAsString());
    await _expectProcessGone(parentPid);
    await _expectProcessGone(childPid);
  });
}

Future<void> _expectProcessGone(int processId) async {
  for (var attempt = 0; attempt < 40; attempt++) {
    final result = await Process.run('ps', ['-p', '$processId', '-o', 'pid=']);
    if ((result.stdout as String).trim().isEmpty) {
      return;
    }
    await Future<void>.delayed(const Duration(milliseconds: 50));
  }
  fail('Process $processId survived Cargo process-group cleanup.');
}

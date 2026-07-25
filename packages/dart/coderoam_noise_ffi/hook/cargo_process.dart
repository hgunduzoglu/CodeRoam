import 'dart:async';
import 'dart:ffi';
import 'dart:io';

final class CargoProcessTimeout implements Exception {
  const CargoProcessTimeout();
}

final class CargoProcessCleanupFailure implements Exception {
  const CargoProcessCleanupFailure();
}

Future<int> runCargoProcess({
  required String executable,
  required List<String> arguments,
  required String workingDirectory,
  required Map<String, String> environment,
  required Uri runnerScript,
  required Uri scratchDirectory,
  Duration timeout = const Duration(minutes: 5),
  Duration terminationGrace = const Duration(seconds: 2),
  Duration killGrace = const Duration(seconds: 2),
  Duration pollInterval = const Duration(milliseconds: 25),
}) async {
  if (!Platform.isMacOS && !Platform.isLinux) {
    throw UnsupportedError('Cargo process isolation requires a POSIX host.');
  }

  await Directory.fromUri(scratchDirectory).create(recursive: true);
  final token = '$pid-${DateTime.now().microsecondsSinceEpoch}';
  final statusFile = File.fromUri(
    scratchDirectory.resolve('cargo-$token.status'),
  );
  final startGate = File.fromUri(
    scratchDirectory.resolve('cargo-$token.start'),
  );
  final process = await Process.start(
    Platform.resolvedExecutable,
    [
      runnerScript.toFilePath(),
      statusFile.path,
      startGate.path,
      executable,
      ...arguments,
    ],
    workingDirectory: workingDirectory,
    environment: environment,
    mode: ProcessStartMode.detachedWithStdio,
  );
  await process.stdin.close();
  final stdoutSubscription = process.stdout.listen(stdout.add);
  final stderrSubscription = process.stderr.listen(stderr.add);

  final processGroupId = _getProcessGroupId(process.pid);
  if (processGroupId <= 0) {
    process.kill(ProcessSignal.sigkill);
    await stdoutSubscription.cancel();
    await stderrSubscription.cancel();
    throw const CargoProcessCleanupFailure();
  }
  try {
    await startGate.writeAsString('start', flush: true);
    final deadline = DateTime.now().add(timeout);
    while (DateTime.now().isBefore(deadline)) {
      if (await statusFile.exists()) {
        final exitCode = int.tryParse(await statusFile.readAsString());
        if (exitCode == null) {
          await _terminateProcessGroup(
            processGroupId,
            terminationGrace,
            killGrace,
            pollInterval,
          );
          throw const CargoProcessCleanupFailure();
        }
        if (!await _waitForProcessGroupExit(
          processGroupId,
          terminationGrace,
          pollInterval,
        )) {
          await _terminateProcessGroup(
            processGroupId,
            terminationGrace,
            killGrace,
            pollInterval,
          );
          throw const CargoProcessCleanupFailure();
        }
        return exitCode;
      }
      await Future<void>.delayed(pollInterval);
    }

    await _terminateProcessGroup(
      processGroupId,
      terminationGrace,
      killGrace,
      pollInterval,
    );
    throw const CargoProcessTimeout();
  } on CargoProcessTimeout {
    rethrow;
  } on CargoProcessCleanupFailure {
    rethrow;
  } catch (_) {
    await _terminateProcessGroup(
      processGroupId,
      terminationGrace,
      killGrace,
      pollInterval,
    );
    rethrow;
  } finally {
    await stdoutSubscription.cancel();
    await stderrSubscription.cancel();
    if (await statusFile.exists()) {
      await statusFile.delete();
    }
    if (await startGate.exists()) {
      await startGate.delete();
    }
  }
}

Future<void> _terminateProcessGroup(
  int processGroupId,
  Duration terminationGrace,
  Duration killGrace,
  Duration pollInterval,
) async {
  _signalProcessGroup(processGroupId, ProcessSignal.sigterm);
  if (await _waitForProcessGroupExit(
    processGroupId,
    terminationGrace,
    pollInterval,
  )) {
    return;
  }

  _signalProcessGroup(processGroupId, ProcessSignal.sigkill);
  if (!await _waitForProcessGroupExit(
    processGroupId,
    killGrace,
    pollInterval,
  )) {
    throw const CargoProcessCleanupFailure();
  }
}

Future<bool> _waitForProcessGroupExit(
  int processGroupId,
  Duration timeout,
  Duration pollInterval,
) async {
  final deadline = DateTime.now().add(timeout);
  while (_processGroupExists(processGroupId)) {
    if (!DateTime.now().isBefore(deadline)) {
      return false;
    }
    await Future<void>.delayed(pollInterval);
  }
  return true;
}

typedef _GetProcessGroupIdNative = Int32 Function(Int32);
typedef _GetProcessGroupIdDart = int Function(int);
typedef _KillProcessGroupNative = Int32 Function(Int32, Int32);
typedef _KillProcessGroupDart = int Function(int, int);

final _getProcessGroupId = DynamicLibrary.process()
    .lookupFunction<_GetProcessGroupIdNative, _GetProcessGroupIdDart>(
      'getpgid',
    );
final _killProcessGroup = DynamicLibrary.process()
    .lookupFunction<_KillProcessGroupNative, _KillProcessGroupDart>('killpg');

bool _processGroupExists(int processGroupId) {
  return _killProcessGroup(processGroupId, 0) == 0;
}

void _signalProcessGroup(int processGroupId, ProcessSignal signal) {
  _killProcessGroup(processGroupId, signal.signalNumber);
}

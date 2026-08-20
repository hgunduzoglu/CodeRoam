import 'dart:async';
import 'dart:io';

Future<void> main(List<String> arguments) async {
  if (arguments.length < 4) {
    stderr.writeln('Cargo runner received an invalid invocation.');
    exitCode = 64;
    return;
  }

  final statusFile = File(arguments[0]);
  final startGate = File(arguments[1]);
  final executable = arguments[2];
  final executableArguments = arguments.sublist(3);
  final gateDeadline = DateTime.now().add(const Duration(seconds: 30));
  while (!await startGate.exists()) {
    if (!DateTime.now().isBefore(gateDeadline)) {
      await _writeExitStatus(statusFile, 124);
      return;
    }
    await Future<void>.delayed(const Duration(milliseconds: 5));
  }

  var childExitCode = 127;
  try {
    final child = await Process.start(
      executable,
      executableArguments,
      mode: ProcessStartMode.inheritStdio,
    );
    childExitCode = await child.exitCode;
  } on ProcessException {
    stderr.writeln('Cargo runner could not start the requested executable.');
  }
  await _writeExitStatus(statusFile, childExitCode);
}

Future<void> _writeExitStatus(File statusFile, int childExitCode) async {
  final temporaryFile = File('${statusFile.path}.tmp');
  await temporaryFile.writeAsString('$childExitCode', flush: true);
  await temporaryFile.rename(statusFile.path);
}

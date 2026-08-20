import 'dart:io';

Future<void> main(List<String> arguments) async {
  final childScript = arguments[0];
  final parentPidFile = File(arguments[1]);
  final childPidFile = File(arguments[2]);
  await parentPidFile.writeAsString('$pid', flush: true);
  await Process.start(Platform.resolvedExecutable, [
    childScript,
    childPidFile.path,
  ], mode: ProcessStartMode.inheritStdio);
  while (true) {
    await Future<void>.delayed(const Duration(seconds: 1));
  }
}

import 'dart:io';

Future<void> main(List<String> arguments) async {
  await File(arguments[0]).writeAsString('$pid', flush: true);
  while (true) {
    await Future<void>.delayed(const Duration(seconds: 1));
  }
}

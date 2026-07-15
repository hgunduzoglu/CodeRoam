import 'package:coderoam/features/workspace/presentation/touch_spike_shell.dart';
import 'package:flutter/material.dart';

void main() {
  runApp(const CodeRoamApp());
}

class CodeRoamApp extends StatelessWidget {
  const CodeRoamApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'CodeRoam',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF635BFF),
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: const TouchSpikeShell(),
    );
  }
}

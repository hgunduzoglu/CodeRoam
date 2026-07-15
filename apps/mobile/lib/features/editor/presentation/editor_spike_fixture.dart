enum EditorSpikeFixture { sample, largeFile }

const sampleEditorSpikeDocument = '''
// Document supplied by Flutter through the typed bridge.

function greet(name: string): string {
  return `Welcome, \${name}`;
}

console.log(greet("CodeRoam"));
''';

const largeEditorSpikeLineCount = 10000;

String editorSpikeDocument(EditorSpikeFixture fixture) {
  return switch (fixture) {
    EditorSpikeFixture.sample => sampleEditorSpikeDocument,
    EditorSpikeFixture.largeFile => _buildLargeEditorSpikeDocument(),
  };
}

String _buildLargeEditorSpikeDocument() {
  final document = StringBuffer();

  for (var line = 1; line <= largeEditorSpikeLineCount; line += 1) {
    document.writeln(
      'export const fixtureLine${line.toString().padLeft(5, '0')} = $line;',
    );
  }

  return document.toString();
}

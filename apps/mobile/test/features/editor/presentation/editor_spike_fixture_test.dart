import 'dart:convert';

import 'package:coderoam/features/editor/presentation/editor_spike_fixture.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('large editor fixture contains exactly 10,000 searchable lines', () {
    final document = editorSpikeDocument(EditorSpikeFixture.largeFile);
    final lines = const LineSplitter().convert(document);

    expect(lines, hasLength(largeEditorSpikeLineCount));
    expect(lines.first, 'export const fixtureLine00001 = 1;');
    expect(lines.last, 'export const fixtureLine10000 = 10000;');
  });

  test('sample fixture contains the diagnostic exercise', () {
    expect(sampleEditorSpikeDocument, contains('console.log'));
  });
}

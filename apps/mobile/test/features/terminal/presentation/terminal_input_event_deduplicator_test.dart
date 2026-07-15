import 'package:coderoam/features/terminal/presentation/terminal_input_event_deduplicator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('accepts delayed unique events and rejects replayed duplicates', () {
    final deduplicator = TerminalInputEventDeduplicator();

    expect(deduplicator.beginStream('page-a'), isTrue);
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:2'),
      isTrue,
    );
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:1'),
      isTrue,
    );
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:1'),
      isFalse,
    );
  });

  test('changes streams only after an explicit page reset', () {
    final deduplicator = TerminalInputEventDeduplicator();

    expect(deduplicator.beginStream('page-a'), isTrue);
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:1'),
      isTrue,
    );

    expect(deduplicator.beginStream('page-b'), isFalse);
    expect(
      deduplicator.accept(streamId: 'page-b', eventId: 'page-b:1'),
      isFalse,
    );

    deduplicator.reset();
    expect(deduplicator.beginStream('page-a'), isFalse);
    expect(deduplicator.beginStream('page-b'), isTrue);
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:2'),
      isFalse,
    );
    expect(
      deduplicator.accept(streamId: 'page-b', eventId: 'page-b:1'),
      isTrue,
    );
    expect(deduplicator.beginStream('page-a'), isFalse);
  });

  test('bounds remembered event IDs', () {
    final deduplicator = TerminalInputEventDeduplicator(
      maxRememberedEventIds: 2,
    );

    expect(deduplicator.beginStream('page-a'), isTrue);
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:1'),
      isTrue,
    );
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:2'),
      isTrue,
    );
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:3'),
      isTrue,
    );
    expect(
      deduplicator.accept(streamId: 'page-a', eventId: 'page-a:1'),
      isTrue,
    );
  });

  test('rejects invalid configuration and malformed IDs', () {
    expect(
      () => TerminalInputEventDeduplicator(maxRememberedEventIds: 0),
      throwsArgumentError,
    );
    final deduplicator = TerminalInputEventDeduplicator();
    expect(deduplicator.beginStream(''), isFalse);
    expect(deduplicator.beginStream('a' * 65), isFalse);
    expect(deduplicator.beginStream('page-a'), isTrue);

    for (final eventId in [
      '',
      'other:1',
      'page-a:0',
      'page-a:01',
      'page-a:x',
      'page-a:${'1' * 17}',
    ]) {
      expect(
        deduplicator.accept(streamId: 'page-a', eventId: eventId),
        isFalse,
        reason: eventId,
      );
    }
  });
}

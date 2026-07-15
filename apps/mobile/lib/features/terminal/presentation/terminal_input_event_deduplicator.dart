import 'dart:collection';

class TerminalInputEventDeduplicator {
  TerminalInputEventDeduplicator({this.maxRememberedEventIds = 256}) {
    if (maxRememberedEventIds < 1) {
      throw ArgumentError.value(
        maxRememberedEventIds,
        'maxRememberedEventIds',
        'Must be greater than zero.',
      );
    }
  }

  final int maxRememberedEventIds;

  static const int _maxStreamIdLength = 64;
  static const int _maxSequenceDigits = 16;
  static const int _maxRetiredStreamIds = 8;

  final Set<String> _rememberedEventIds = {};
  final Queue<String> _eventIdOrder = Queue<String>();
  final Set<String> _retiredStreamIds = {};
  final Queue<String> _retiredStreamIdOrder = Queue<String>();
  String? _activeStreamId;

  bool beginStream(String streamId) {
    if (streamId.isEmpty ||
        streamId.length > _maxStreamIdLength ||
        _retiredStreamIds.contains(streamId)) {
      return false;
    }

    if (_activeStreamId == null) {
      _activeStreamId = streamId;
      _clearRememberedEvents();
      return true;
    }

    return _activeStreamId == streamId;
  }

  void reset() {
    final activeStreamId = _activeStreamId;
    if (activeStreamId != null) {
      _retiredStreamIds.add(activeStreamId);
      _retiredStreamIdOrder.addLast(activeStreamId);

      if (_retiredStreamIdOrder.length > _maxRetiredStreamIds) {
        _retiredStreamIds.remove(_retiredStreamIdOrder.removeFirst());
      }
    }

    _activeStreamId = null;
    _clearRememberedEvents();
  }

  bool accept({required String streamId, required String eventId}) {
    if (streamId.isEmpty ||
        streamId != _activeStreamId ||
        !_isEventIdForStream(streamId: streamId, eventId: eventId) ||
        _rememberedEventIds.contains(eventId)) {
      return false;
    }

    _rememberedEventIds.add(eventId);
    _eventIdOrder.addLast(eventId);

    if (_eventIdOrder.length > maxRememberedEventIds) {
      _rememberedEventIds.remove(_eventIdOrder.removeFirst());
    }

    return true;
  }

  bool isActiveStream(String streamId) {
    return streamId.isNotEmpty && streamId == _activeStreamId;
  }

  bool _isEventIdForStream({
    required String streamId,
    required String eventId,
  }) {
    final prefix = '$streamId:';
    if (!eventId.startsWith(prefix)) {
      return false;
    }

    final sequenceText = eventId.substring(prefix.length);
    if (sequenceText.isEmpty || sequenceText.length > _maxSequenceDigits) {
      return false;
    }

    final sequence = int.tryParse(sequenceText);
    return sequence != null &&
        sequence > 0 &&
        sequence.toString() == sequenceText;
  }

  void _clearRememberedEvents() {
    _rememberedEventIds.clear();
    _eventIdOrder.clear();
  }
}

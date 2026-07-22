import 'dart:convert';

const _maxJsonNestingDepth = 64;

Object? decodeStrictJson(String source) {
  try {
    _JsonDuplicateKeyScanner(source).scan();
    return jsonDecode(source);
  } on FormatException {
    throw const FormatException('Control-plane JSON is invalid.');
  }
}

final class _JsonDuplicateKeyScanner {
  _JsonDuplicateKeyScanner(this._source);

  final String _source;
  int _index = 0;

  void scan() {
    _skipWhitespace();
    _scanValue(0);
    _skipWhitespace();
    if (_index != _source.length) {
      throw const FormatException('Trailing JSON data.');
    }
  }

  void _scanValue(int depth) {
    if (depth > _maxJsonNestingDepth || _index >= _source.length) {
      throw const FormatException('JSON value is invalid.');
    }
    switch (_source.codeUnitAt(_index)) {
      case 0x7b:
        _scanObject(depth + 1);
        return;
      case 0x5b:
        _scanArray(depth + 1);
        return;
      case 0x22:
        _scanString();
        return;
      default:
        _scanPrimitive();
        return;
    }
  }

  void _scanObject(int depth) {
    _index++;
    _skipWhitespace();
    if (_consume(0x7d)) {
      return;
    }
    final keys = <String>{};
    while (true) {
      if (_index >= _source.length || _source.codeUnitAt(_index) != 0x22) {
        throw const FormatException('JSON object key is invalid.');
      }
      final start = _index;
      _scanString();
      final decodedKey = jsonDecode(_source.substring(start, _index));
      if (decodedKey is! String || !keys.add(decodedKey)) {
        throw const FormatException('Duplicate JSON object key.');
      }
      _skipWhitespace();
      if (!_consume(0x3a)) {
        throw const FormatException('JSON object separator is invalid.');
      }
      _skipWhitespace();
      _scanValue(depth);
      _skipWhitespace();
      if (_consume(0x7d)) {
        return;
      }
      if (!_consume(0x2c)) {
        throw const FormatException('JSON object delimiter is invalid.');
      }
      _skipWhitespace();
    }
  }

  void _scanArray(int depth) {
    _index++;
    _skipWhitespace();
    if (_consume(0x5d)) {
      return;
    }
    while (true) {
      _scanValue(depth);
      _skipWhitespace();
      if (_consume(0x5d)) {
        return;
      }
      if (!_consume(0x2c)) {
        throw const FormatException('JSON array delimiter is invalid.');
      }
      _skipWhitespace();
    }
  }

  void _scanString() {
    _index++;
    while (_index < _source.length) {
      final codeUnit = _source.codeUnitAt(_index++);
      if (codeUnit == 0x22) {
        return;
      }
      if (codeUnit < 0x20) {
        throw const FormatException('JSON string contains a control value.');
      }
      if (codeUnit != 0x5c) {
        continue;
      }
      if (_index >= _source.length) {
        throw const FormatException('JSON string escape is incomplete.');
      }
      final escape = _source.codeUnitAt(_index++);
      if (escape == 0x75) {
        if (_index + 4 > _source.length) {
          throw const FormatException('JSON unicode escape is incomplete.');
        }
        _index += 4;
      }
    }
    throw const FormatException('JSON string is incomplete.');
  }

  void _scanPrimitive() {
    final start = _index;
    while (_index < _source.length) {
      final codeUnit = _source.codeUnitAt(_index);
      if (_isWhitespace(codeUnit) ||
          codeUnit == 0x2c ||
          codeUnit == 0x5d ||
          codeUnit == 0x7d) {
        break;
      }
      _index++;
    }
    if (_index == start) {
      throw const FormatException('JSON primitive is invalid.');
    }
  }

  void _skipWhitespace() {
    while (_index < _source.length &&
        _isWhitespace(_source.codeUnitAt(_index))) {
      _index++;
    }
  }

  bool _consume(int codeUnit) {
    if (_index >= _source.length || _source.codeUnitAt(_index) != codeUnit) {
      return false;
    }
    _index++;
    return true;
  }
}

bool _isWhitespace(int codeUnit) =>
    codeUnit == 0x20 ||
    codeUnit == 0x09 ||
    codeUnit == 0x0a ||
    codeUnit == 0x0d;

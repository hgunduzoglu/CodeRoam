// This is a generated file - do not edit.
//
// Generated from coderoam/common/v1/envelope.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class ChannelKind extends $pb.ProtobufEnum {
  static const ChannelKind CHANNEL_KIND_UNSPECIFIED =
      ChannelKind._(0, _omitEnumNames ? '' : 'CHANNEL_KIND_UNSPECIFIED');
  static const ChannelKind CHANNEL_KIND_CONTROL =
      ChannelKind._(1, _omitEnumNames ? '' : 'CHANNEL_KIND_CONTROL');
  static const ChannelKind CHANNEL_KIND_FILESYSTEM =
      ChannelKind._(2, _omitEnumNames ? '' : 'CHANNEL_KIND_FILESYSTEM');
  static const ChannelKind CHANNEL_KIND_EDITOR =
      ChannelKind._(3, _omitEnumNames ? '' : 'CHANNEL_KIND_EDITOR');
  static const ChannelKind CHANNEL_KIND_TERMINAL =
      ChannelKind._(4, _omitEnumNames ? '' : 'CHANNEL_KIND_TERMINAL');
  static const ChannelKind CHANNEL_KIND_GIT =
      ChannelKind._(5, _omitEnumNames ? '' : 'CHANNEL_KIND_GIT');
  static const ChannelKind CHANNEL_KIND_LSP =
      ChannelKind._(6, _omitEnumNames ? '' : 'CHANNEL_KIND_LSP');
  static const ChannelKind CHANNEL_KIND_AI =
      ChannelKind._(7, _omitEnumNames ? '' : 'CHANNEL_KIND_AI');
  static const ChannelKind CHANNEL_KIND_LOGS =
      ChannelKind._(8, _omitEnumNames ? '' : 'CHANNEL_KIND_LOGS');
  static const ChannelKind CHANNEL_KIND_PORT_FORWARD =
      ChannelKind._(9, _omitEnumNames ? '' : 'CHANNEL_KIND_PORT_FORWARD');

  static const $core.List<ChannelKind> values = <ChannelKind>[
    CHANNEL_KIND_UNSPECIFIED,
    CHANNEL_KIND_CONTROL,
    CHANNEL_KIND_FILESYSTEM,
    CHANNEL_KIND_EDITOR,
    CHANNEL_KIND_TERMINAL,
    CHANNEL_KIND_GIT,
    CHANNEL_KIND_LSP,
    CHANNEL_KIND_AI,
    CHANNEL_KIND_LOGS,
    CHANNEL_KIND_PORT_FORWARD,
  ];

  static final $core.List<ChannelKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 9);
  static ChannelKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ChannelKind._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');

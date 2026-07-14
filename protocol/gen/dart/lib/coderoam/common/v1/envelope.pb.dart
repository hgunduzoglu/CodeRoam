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

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'envelope.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'envelope.pbenum.dart';

class EncryptedFrame extends $pb.GeneratedMessage {
  factory EncryptedFrame({
    $core.int? protocolVersion,
    $core.List<$core.int>? sessionId,
    ChannelKind? channel,
    $core.int? streamId,
    $fixnum.Int64? sequence,
    $fixnum.Int64? acknowledgement,
    $core.int? flags,
    $core.List<$core.int>? ciphertext,
  }) {
    final result = create();
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (sessionId != null) result.sessionId = sessionId;
    if (channel != null) result.channel = channel;
    if (streamId != null) result.streamId = streamId;
    if (sequence != null) result.sequence = sequence;
    if (acknowledgement != null) result.acknowledgement = acknowledgement;
    if (flags != null) result.flags = flags;
    if (ciphertext != null) result.ciphertext = ciphertext;
    return result;
  }

  EncryptedFrame._();

  factory EncryptedFrame.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EncryptedFrame.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EncryptedFrame',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.common.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'sessionId', $pb.PbFieldType.OY)
    ..aE<ChannelKind>(3, _omitFieldNames ? '' : 'channel',
        enumValues: ChannelKind.values)
    ..aI(4, _omitFieldNames ? '' : 'streamId', fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'sequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'acknowledgement', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(7, _omitFieldNames ? '' : 'flags', fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'ciphertext', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EncryptedFrame clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EncryptedFrame copyWith(void Function(EncryptedFrame) updates) =>
      super.copyWith((message) => updates(message as EncryptedFrame))
          as EncryptedFrame;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EncryptedFrame create() => EncryptedFrame._();
  @$core.override
  EncryptedFrame createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EncryptedFrame getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EncryptedFrame>(create);
  static EncryptedFrame? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get protocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set protocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get sessionId => $_getN(1);
  @$pb.TagNumber(2)
  set sessionId($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  ChannelKind get channel => $_getN(2);
  @$pb.TagNumber(3)
  set channel(ChannelKind value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasChannel() => $_has(2);
  @$pb.TagNumber(3)
  void clearChannel() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get streamId => $_getIZ(3);
  @$pb.TagNumber(4)
  set streamId($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStreamId() => $_has(3);
  @$pb.TagNumber(4)
  void clearStreamId() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get sequence => $_getI64(4);
  @$pb.TagNumber(5)
  set sequence($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasSequence() => $_has(4);
  @$pb.TagNumber(5)
  void clearSequence() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get acknowledgement => $_getI64(5);
  @$pb.TagNumber(6)
  set acknowledgement($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAcknowledgement() => $_has(5);
  @$pb.TagNumber(6)
  void clearAcknowledgement() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get flags => $_getIZ(6);
  @$pb.TagNumber(7)
  set flags($core.int value) => $_setUnsignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasFlags() => $_has(6);
  @$pb.TagNumber(7)
  void clearFlags() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.List<$core.int> get ciphertext => $_getN(7);
  @$pb.TagNumber(8)
  set ciphertext($core.List<$core.int> value) => $_setBytes(7, value);
  @$pb.TagNumber(8)
  $core.bool hasCiphertext() => $_has(7);
  @$pb.TagNumber(8)
  void clearCiphertext() => $_clearField(8);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');

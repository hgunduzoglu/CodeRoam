// This is a generated file - do not edit.
//
// Generated from coderoam/control/v1/session.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class ResumeRequest extends $pb.GeneratedMessage {
  factory ResumeRequest({
    $core.List<$core.int>? previousSessionId,
    $core.Iterable<$core.MapEntry<$core.int, $fixnum.Int64>>?
        acknowledgedSequences,
  }) {
    final result = create();
    if (previousSessionId != null) result.previousSessionId = previousSessionId;
    if (acknowledgedSequences != null)
      result.acknowledgedSequences.addEntries(acknowledgedSequences);
    return result;
  }

  ResumeRequest._();

  factory ResumeRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResumeRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResumeRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.control.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'previousSessionId', $pb.PbFieldType.OY)
    ..m<$core.int, $fixnum.Int64>(
        2, _omitFieldNames ? '' : 'acknowledgedSequences',
        entryClassName: 'ResumeRequest.AcknowledgedSequencesEntry',
        keyFieldType: $pb.PbFieldType.OU3,
        valueFieldType: $pb.PbFieldType.OU6,
        packageName: const $pb.PackageName('coderoam.control.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeRequest copyWith(void Function(ResumeRequest) updates) =>
      super.copyWith((message) => updates(message as ResumeRequest))
          as ResumeRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResumeRequest create() => ResumeRequest._();
  @$core.override
  ResumeRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResumeRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResumeRequest>(create);
  static ResumeRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get previousSessionId => $_getN(0);
  @$pb.TagNumber(1)
  set previousSessionId($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPreviousSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPreviousSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbMap<$core.int, $fixnum.Int64> get acknowledgedSequences => $_getMap(1);
}

class ResumeAccepted extends $pb.GeneratedMessage {
  factory ResumeAccepted({
    $core.List<$core.int>? sessionId,
    $core.Iterable<$core.MapEntry<$core.int, $fixnum.Int64>>? replayStarts,
  }) {
    final result = create();
    if (sessionId != null) result.sessionId = sessionId;
    if (replayStarts != null) result.replayStarts.addEntries(replayStarts);
    return result;
  }

  ResumeAccepted._();

  factory ResumeAccepted.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResumeAccepted.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResumeAccepted',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.control.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'sessionId', $pb.PbFieldType.OY)
    ..m<$core.int, $fixnum.Int64>(2, _omitFieldNames ? '' : 'replayStarts',
        entryClassName: 'ResumeAccepted.ReplayStartsEntry',
        keyFieldType: $pb.PbFieldType.OU3,
        valueFieldType: $pb.PbFieldType.OU6,
        packageName: const $pb.PackageName('coderoam.control.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeAccepted clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeAccepted copyWith(void Function(ResumeAccepted) updates) =>
      super.copyWith((message) => updates(message as ResumeAccepted))
          as ResumeAccepted;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResumeAccepted create() => ResumeAccepted._();
  @$core.override
  ResumeAccepted createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResumeAccepted getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResumeAccepted>(create);
  static ResumeAccepted? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get sessionId => $_getN(0);
  @$pb.TagNumber(1)
  set sessionId($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSessionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearSessionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbMap<$core.int, $fixnum.Int64> get replayStarts => $_getMap(1);
}

class ResumeUnavailable extends $pb.GeneratedMessage {
  factory ResumeUnavailable({
    $core.String? reason,
  }) {
    final result = create();
    if (reason != null) result.reason = reason;
    return result;
  }

  ResumeUnavailable._();

  factory ResumeUnavailable.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResumeUnavailable.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResumeUnavailable',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeUnavailable clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeUnavailable copyWith(void Function(ResumeUnavailable) updates) =>
      super.copyWith((message) => updates(message as ResumeUnavailable))
          as ResumeUnavailable;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResumeUnavailable create() => ResumeUnavailable._();
  @$core.override
  ResumeUnavailable createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResumeUnavailable getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResumeUnavailable>(create);
  static ResumeUnavailable? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reason => $_getSZ(0);
  @$pb.TagNumber(1)
  set reason($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReason() => $_has(0);
  @$pb.TagNumber(1)
  void clearReason() => $_clearField(1);
}

class Heartbeat extends $pb.GeneratedMessage {
  factory Heartbeat({
    $fixnum.Int64? sentAtUnixMillis,
  }) {
    final result = create();
    if (sentAtUnixMillis != null) result.sentAtUnixMillis = sentAtUnixMillis;
    return result;
  }

  Heartbeat._();

  factory Heartbeat.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Heartbeat.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Heartbeat',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.control.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'sentAtUnixMillis')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Heartbeat clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Heartbeat copyWith(void Function(Heartbeat) updates) =>
      super.copyWith((message) => updates(message as Heartbeat)) as Heartbeat;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Heartbeat create() => Heartbeat._();
  @$core.override
  Heartbeat createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Heartbeat getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Heartbeat>(create);
  static Heartbeat? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get sentAtUnixMillis => $_getI64(0);
  @$pb.TagNumber(1)
  set sentAtUnixMillis($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSentAtUnixMillis() => $_has(0);
  @$pb.TagNumber(1)
  void clearSentAtUnixMillis() => $_clearField(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');

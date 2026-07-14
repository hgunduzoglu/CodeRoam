// This is a generated file - do not edit.
//
// Generated from coderoam/agent/v1/terminal.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class OpenTerminalRequest extends $pb.GeneratedMessage {
  factory OpenTerminalRequest({
    $core.String? projectId,
    $core.int? columns,
    $core.int? rows,
  }) {
    final result = create();
    if (projectId != null) result.projectId = projectId;
    if (columns != null) result.columns = columns;
    if (rows != null) result.rows = rows;
    return result;
  }

  OpenTerminalRequest._();

  factory OpenTerminalRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OpenTerminalRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OpenTerminalRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'projectId')
    ..aI(2, _omitFieldNames ? '' : 'columns', fieldType: $pb.PbFieldType.OU3)
    ..aI(3, _omitFieldNames ? '' : 'rows', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenTerminalRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OpenTerminalRequest copyWith(void Function(OpenTerminalRequest) updates) =>
      super.copyWith((message) => updates(message as OpenTerminalRequest))
          as OpenTerminalRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OpenTerminalRequest create() => OpenTerminalRequest._();
  @$core.override
  OpenTerminalRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OpenTerminalRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OpenTerminalRequest>(create);
  static OpenTerminalRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get projectId => $_getSZ(0);
  @$pb.TagNumber(1)
  set projectId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProjectId() => $_has(0);
  @$pb.TagNumber(1)
  void clearProjectId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get columns => $_getIZ(1);
  @$pb.TagNumber(2)
  set columns($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasColumns() => $_has(1);
  @$pb.TagNumber(2)
  void clearColumns() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get rows => $_getIZ(2);
  @$pb.TagNumber(3)
  set rows($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRows() => $_has(2);
  @$pb.TagNumber(3)
  void clearRows() => $_clearField(3);
}

class TerminalInput extends $pb.GeneratedMessage {
  factory TerminalInput({
    $core.List<$core.int>? data,
  }) {
    final result = create();
    if (data != null) result.data = data;
    return result;
  }

  TerminalInput._();

  factory TerminalInput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalInput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalInput',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalInput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalInput copyWith(void Function(TerminalInput) updates) =>
      super.copyWith((message) => updates(message as TerminalInput))
          as TerminalInput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalInput create() => TerminalInput._();
  @$core.override
  TerminalInput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalInput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalInput>(create);
  static TerminalInput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get data => $_getN(0);
  @$pb.TagNumber(1)
  set data($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasData() => $_has(0);
  @$pb.TagNumber(1)
  void clearData() => $_clearField(1);
}

class TerminalOutput extends $pb.GeneratedMessage {
  factory TerminalOutput({
    $core.List<$core.int>? data,
  }) {
    final result = create();
    if (data != null) result.data = data;
    return result;
  }

  TerminalOutput._();

  factory TerminalOutput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TerminalOutput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TerminalOutput',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'data', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalOutput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TerminalOutput copyWith(void Function(TerminalOutput) updates) =>
      super.copyWith((message) => updates(message as TerminalOutput))
          as TerminalOutput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TerminalOutput create() => TerminalOutput._();
  @$core.override
  TerminalOutput createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TerminalOutput getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TerminalOutput>(create);
  static TerminalOutput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get data => $_getN(0);
  @$pb.TagNumber(1)
  set data($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasData() => $_has(0);
  @$pb.TagNumber(1)
  void clearData() => $_clearField(1);
}

class ResizeTerminal extends $pb.GeneratedMessage {
  factory ResizeTerminal({
    $core.int? columns,
    $core.int? rows,
  }) {
    final result = create();
    if (columns != null) result.columns = columns;
    if (rows != null) result.rows = rows;
    return result;
  }

  ResizeTerminal._();

  factory ResizeTerminal.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ResizeTerminal.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResizeTerminal',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'columns', fieldType: $pb.PbFieldType.OU3)
    ..aI(2, _omitFieldNames ? '' : 'rows', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResizeTerminal clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResizeTerminal copyWith(void Function(ResizeTerminal) updates) =>
      super.copyWith((message) => updates(message as ResizeTerminal))
          as ResizeTerminal;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResizeTerminal create() => ResizeTerminal._();
  @$core.override
  ResizeTerminal createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ResizeTerminal getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResizeTerminal>(create);
  static ResizeTerminal? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get columns => $_getIZ(0);
  @$pb.TagNumber(1)
  set columns($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasColumns() => $_has(0);
  @$pb.TagNumber(1)
  void clearColumns() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get rows => $_getIZ(1);
  @$pb.TagNumber(2)
  set rows($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRows() => $_has(1);
  @$pb.TagNumber(2)
  void clearRows() => $_clearField(2);
}

class CloseTerminal extends $pb.GeneratedMessage {
  factory CloseTerminal({
    $core.String? reason,
  }) {
    final result = create();
    if (reason != null) result.reason = reason;
    return result;
  }

  CloseTerminal._();

  factory CloseTerminal.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloseTerminal.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloseTerminal',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloseTerminal clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloseTerminal copyWith(void Function(CloseTerminal) updates) =>
      super.copyWith((message) => updates(message as CloseTerminal))
          as CloseTerminal;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloseTerminal create() => CloseTerminal._();
  @$core.override
  CloseTerminal createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloseTerminal getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloseTerminal>(create);
  static CloseTerminal? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reason => $_getSZ(0);
  @$pb.TagNumber(1)
  set reason($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReason() => $_has(0);
  @$pb.TagNumber(1)
  void clearReason() => $_clearField(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');

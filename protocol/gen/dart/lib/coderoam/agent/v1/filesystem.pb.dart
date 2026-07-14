// This is a generated file - do not edit.
//
// Generated from coderoam/agent/v1/filesystem.proto.

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

class ReadFileRequest extends $pb.GeneratedMessage {
  factory ReadFileRequest({
    $core.String? projectId,
    $core.String? relativePath,
  }) {
    final result = create();
    if (projectId != null) result.projectId = projectId;
    if (relativePath != null) result.relativePath = relativePath;
    return result;
  }

  ReadFileRequest._();

  factory ReadFileRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReadFileRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReadFileRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'projectId')
    ..aOS(2, _omitFieldNames ? '' : 'relativePath')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadFileRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadFileRequest copyWith(void Function(ReadFileRequest) updates) =>
      super.copyWith((message) => updates(message as ReadFileRequest))
          as ReadFileRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReadFileRequest create() => ReadFileRequest._();
  @$core.override
  ReadFileRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ReadFileRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReadFileRequest>(create);
  static ReadFileRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get projectId => $_getSZ(0);
  @$pb.TagNumber(1)
  set projectId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProjectId() => $_has(0);
  @$pb.TagNumber(1)
  void clearProjectId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get relativePath => $_getSZ(1);
  @$pb.TagNumber(2)
  set relativePath($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRelativePath() => $_has(1);
  @$pb.TagNumber(2)
  void clearRelativePath() => $_clearField(2);
}

class ReadFileResponse extends $pb.GeneratedMessage {
  factory ReadFileResponse({
    $core.List<$core.int>? content,
    $core.String? contentHash,
    $fixnum.Int64? version,
    $core.bool? isBinary,
  }) {
    final result = create();
    if (content != null) result.content = content;
    if (contentHash != null) result.contentHash = contentHash;
    if (version != null) result.version = version;
    if (isBinary != null) result.isBinary = isBinary;
    return result;
  }

  ReadFileResponse._();

  factory ReadFileResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ReadFileResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReadFileResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'content', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'contentHash')
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(4, _omitFieldNames ? '' : 'isBinary')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadFileResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReadFileResponse copyWith(void Function(ReadFileResponse) updates) =>
      super.copyWith((message) => updates(message as ReadFileResponse))
          as ReadFileResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ReadFileResponse create() => ReadFileResponse._();
  @$core.override
  ReadFileResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ReadFileResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ReadFileResponse>(create);
  static ReadFileResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get content => $_getN(0);
  @$pb.TagNumber(1)
  set content($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasContent() => $_has(0);
  @$pb.TagNumber(1)
  void clearContent() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get contentHash => $_getSZ(1);
  @$pb.TagNumber(2)
  set contentHash($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasContentHash() => $_has(1);
  @$pb.TagNumber(2)
  void clearContentHash() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get version => $_getI64(2);
  @$pb.TagNumber(3)
  set version($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get isBinary => $_getBF(3);
  @$pb.TagNumber(4)
  set isBinary($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasIsBinary() => $_has(3);
  @$pb.TagNumber(4)
  void clearIsBinary() => $_clearField(4);
}

class WriteFileRequest extends $pb.GeneratedMessage {
  factory WriteFileRequest({
    $core.String? projectId,
    $core.String? relativePath,
    $core.List<$core.int>? content,
    $core.String? expectedContentHash,
    $fixnum.Int64? expectedVersion,
  }) {
    final result = create();
    if (projectId != null) result.projectId = projectId;
    if (relativePath != null) result.relativePath = relativePath;
    if (content != null) result.content = content;
    if (expectedContentHash != null)
      result.expectedContentHash = expectedContentHash;
    if (expectedVersion != null) result.expectedVersion = expectedVersion;
    return result;
  }

  WriteFileRequest._();

  factory WriteFileRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WriteFileRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WriteFileRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'projectId')
    ..aOS(2, _omitFieldNames ? '' : 'relativePath')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'content', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'expectedContentHash')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'expectedVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteFileRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteFileRequest copyWith(void Function(WriteFileRequest) updates) =>
      super.copyWith((message) => updates(message as WriteFileRequest))
          as WriteFileRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WriteFileRequest create() => WriteFileRequest._();
  @$core.override
  WriteFileRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WriteFileRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WriteFileRequest>(create);
  static WriteFileRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get projectId => $_getSZ(0);
  @$pb.TagNumber(1)
  set projectId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProjectId() => $_has(0);
  @$pb.TagNumber(1)
  void clearProjectId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get relativePath => $_getSZ(1);
  @$pb.TagNumber(2)
  set relativePath($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRelativePath() => $_has(1);
  @$pb.TagNumber(2)
  void clearRelativePath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get content => $_getN(2);
  @$pb.TagNumber(3)
  set content($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasContent() => $_has(2);
  @$pb.TagNumber(3)
  void clearContent() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get expectedContentHash => $_getSZ(3);
  @$pb.TagNumber(4)
  set expectedContentHash($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasExpectedContentHash() => $_has(3);
  @$pb.TagNumber(4)
  void clearExpectedContentHash() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get expectedVersion => $_getI64(4);
  @$pb.TagNumber(5)
  set expectedVersion($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasExpectedVersion() => $_has(4);
  @$pb.TagNumber(5)
  void clearExpectedVersion() => $_clearField(5);
}

class WriteFileResponse extends $pb.GeneratedMessage {
  factory WriteFileResponse({
    $core.String? contentHash,
    $fixnum.Int64? version,
  }) {
    final result = create();
    if (contentHash != null) result.contentHash = contentHash;
    if (version != null) result.version = version;
    return result;
  }

  WriteFileResponse._();

  factory WriteFileResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WriteFileResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WriteFileResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'contentHash')
    ..a<$fixnum.Int64>(2, _omitFieldNames ? '' : 'version', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteFileResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteFileResponse copyWith(void Function(WriteFileResponse) updates) =>
      super.copyWith((message) => updates(message as WriteFileResponse))
          as WriteFileResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WriteFileResponse create() => WriteFileResponse._();
  @$core.override
  WriteFileResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WriteFileResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WriteFileResponse>(create);
  static WriteFileResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get contentHash => $_getSZ(0);
  @$pb.TagNumber(1)
  set contentHash($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasContentHash() => $_has(0);
  @$pb.TagNumber(1)
  void clearContentHash() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get version => $_getI64(1);
  @$pb.TagNumber(2)
  set version($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);
}

class FileConflict extends $pb.GeneratedMessage {
  factory FileConflict({
    $core.String? currentContentHash,
    $fixnum.Int64? currentVersion,
  }) {
    final result = create();
    if (currentContentHash != null)
      result.currentContentHash = currentContentHash;
    if (currentVersion != null) result.currentVersion = currentVersion;
    return result;
  }

  FileConflict._();

  factory FileConflict.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FileConflict.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FileConflict',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.agent.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'currentContentHash')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'currentVersion', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileConflict clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FileConflict copyWith(void Function(FileConflict) updates) =>
      super.copyWith((message) => updates(message as FileConflict))
          as FileConflict;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FileConflict create() => FileConflict._();
  @$core.override
  FileConflict createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FileConflict getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FileConflict>(create);
  static FileConflict? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get currentContentHash => $_getSZ(0);
  @$pb.TagNumber(1)
  set currentContentHash($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCurrentContentHash() => $_has(0);
  @$pb.TagNumber(1)
  void clearCurrentContentHash() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get currentVersion => $_getI64(1);
  @$pb.TagNumber(2)
  set currentVersion($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCurrentVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearCurrentVersion() => $_clearField(2);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');

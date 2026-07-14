// This is a generated file - do not edit.
//
// Generated from coderoam/agent/v1/filesystem.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use readFileRequestDescriptor instead')
const ReadFileRequest$json = {
  '1': 'ReadFileRequest',
  '2': [
    {'1': 'project_id', '3': 1, '4': 1, '5': 9, '10': 'projectId'},
    {'1': 'relative_path', '3': 2, '4': 1, '5': 9, '10': 'relativePath'},
  ],
};

/// Descriptor for `ReadFileRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List readFileRequestDescriptor = $convert.base64Decode(
    'Cg9SZWFkRmlsZVJlcXVlc3QSHQoKcHJvamVjdF9pZBgBIAEoCVIJcHJvamVjdElkEiMKDXJlbG'
    'F0aXZlX3BhdGgYAiABKAlSDHJlbGF0aXZlUGF0aA==');

@$core.Deprecated('Use readFileResponseDescriptor instead')
const ReadFileResponse$json = {
  '1': 'ReadFileResponse',
  '2': [
    {'1': 'content', '3': 1, '4': 1, '5': 12, '10': 'content'},
    {'1': 'content_hash', '3': 2, '4': 1, '5': 9, '10': 'contentHash'},
    {'1': 'version', '3': 3, '4': 1, '5': 4, '10': 'version'},
    {'1': 'is_binary', '3': 4, '4': 1, '5': 8, '10': 'isBinary'},
  ],
};

/// Descriptor for `ReadFileResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List readFileResponseDescriptor = $convert.base64Decode(
    'ChBSZWFkRmlsZVJlc3BvbnNlEhgKB2NvbnRlbnQYASABKAxSB2NvbnRlbnQSIQoMY29udGVudF'
    '9oYXNoGAIgASgJUgtjb250ZW50SGFzaBIYCgd2ZXJzaW9uGAMgASgEUgd2ZXJzaW9uEhsKCWlz'
    'X2JpbmFyeRgEIAEoCFIIaXNCaW5hcnk=');

@$core.Deprecated('Use writeFileRequestDescriptor instead')
const WriteFileRequest$json = {
  '1': 'WriteFileRequest',
  '2': [
    {'1': 'project_id', '3': 1, '4': 1, '5': 9, '10': 'projectId'},
    {'1': 'relative_path', '3': 2, '4': 1, '5': 9, '10': 'relativePath'},
    {'1': 'content', '3': 3, '4': 1, '5': 12, '10': 'content'},
    {
      '1': 'expected_content_hash',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'expectedContentHash'
    },
    {'1': 'expected_version', '3': 5, '4': 1, '5': 4, '10': 'expectedVersion'},
  ],
};

/// Descriptor for `WriteFileRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List writeFileRequestDescriptor = $convert.base64Decode(
    'ChBXcml0ZUZpbGVSZXF1ZXN0Eh0KCnByb2plY3RfaWQYASABKAlSCXByb2plY3RJZBIjCg1yZW'
    'xhdGl2ZV9wYXRoGAIgASgJUgxyZWxhdGl2ZVBhdGgSGAoHY29udGVudBgDIAEoDFIHY29udGVu'
    'dBIyChVleHBlY3RlZF9jb250ZW50X2hhc2gYBCABKAlSE2V4cGVjdGVkQ29udGVudEhhc2gSKQ'
    'oQZXhwZWN0ZWRfdmVyc2lvbhgFIAEoBFIPZXhwZWN0ZWRWZXJzaW9u');

@$core.Deprecated('Use writeFileResponseDescriptor instead')
const WriteFileResponse$json = {
  '1': 'WriteFileResponse',
  '2': [
    {'1': 'content_hash', '3': 1, '4': 1, '5': 9, '10': 'contentHash'},
    {'1': 'version', '3': 2, '4': 1, '5': 4, '10': 'version'},
  ],
};

/// Descriptor for `WriteFileResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List writeFileResponseDescriptor = $convert.base64Decode(
    'ChFXcml0ZUZpbGVSZXNwb25zZRIhCgxjb250ZW50X2hhc2gYASABKAlSC2NvbnRlbnRIYXNoEh'
    'gKB3ZlcnNpb24YAiABKARSB3ZlcnNpb24=');

@$core.Deprecated('Use fileConflictDescriptor instead')
const FileConflict$json = {
  '1': 'FileConflict',
  '2': [
    {
      '1': 'current_content_hash',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'currentContentHash'
    },
    {'1': 'current_version', '3': 2, '4': 1, '5': 4, '10': 'currentVersion'},
  ],
};

/// Descriptor for `FileConflict`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fileConflictDescriptor = $convert.base64Decode(
    'CgxGaWxlQ29uZmxpY3QSMAoUY3VycmVudF9jb250ZW50X2hhc2gYASABKAlSEmN1cnJlbnRDb2'
    '50ZW50SGFzaBInCg9jdXJyZW50X3ZlcnNpb24YAiABKARSDmN1cnJlbnRWZXJzaW9u');

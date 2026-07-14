// This is a generated file - do not edit.
//
// Generated from coderoam/agent/v1/terminal.proto.

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

@$core.Deprecated('Use openTerminalRequestDescriptor instead')
const OpenTerminalRequest$json = {
  '1': 'OpenTerminalRequest',
  '2': [
    {'1': 'project_id', '3': 1, '4': 1, '5': 9, '10': 'projectId'},
    {'1': 'columns', '3': 2, '4': 1, '5': 13, '10': 'columns'},
    {'1': 'rows', '3': 3, '4': 1, '5': 13, '10': 'rows'},
  ],
};

/// Descriptor for `OpenTerminalRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List openTerminalRequestDescriptor = $convert.base64Decode(
    'ChNPcGVuVGVybWluYWxSZXF1ZXN0Eh0KCnByb2plY3RfaWQYASABKAlSCXByb2plY3RJZBIYCg'
    'djb2x1bW5zGAIgASgNUgdjb2x1bW5zEhIKBHJvd3MYAyABKA1SBHJvd3M=');

@$core.Deprecated('Use terminalInputDescriptor instead')
const TerminalInput$json = {
  '1': 'TerminalInput',
  '2': [
    {'1': 'data', '3': 1, '4': 1, '5': 12, '10': 'data'},
  ],
};

/// Descriptor for `TerminalInput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalInputDescriptor =
    $convert.base64Decode('Cg1UZXJtaW5hbElucHV0EhIKBGRhdGEYASABKAxSBGRhdGE=');

@$core.Deprecated('Use terminalOutputDescriptor instead')
const TerminalOutput$json = {
  '1': 'TerminalOutput',
  '2': [
    {'1': 'data', '3': 1, '4': 1, '5': 12, '10': 'data'},
  ],
};

/// Descriptor for `TerminalOutput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List terminalOutputDescriptor =
    $convert.base64Decode('Cg5UZXJtaW5hbE91dHB1dBISCgRkYXRhGAEgASgMUgRkYXRh');

@$core.Deprecated('Use resizeTerminalDescriptor instead')
const ResizeTerminal$json = {
  '1': 'ResizeTerminal',
  '2': [
    {'1': 'columns', '3': 1, '4': 1, '5': 13, '10': 'columns'},
    {'1': 'rows', '3': 2, '4': 1, '5': 13, '10': 'rows'},
  ],
};

/// Descriptor for `ResizeTerminal`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resizeTerminalDescriptor = $convert.base64Decode(
    'Cg5SZXNpemVUZXJtaW5hbBIYCgdjb2x1bW5zGAEgASgNUgdjb2x1bW5zEhIKBHJvd3MYAiABKA'
    '1SBHJvd3M=');

@$core.Deprecated('Use closeTerminalDescriptor instead')
const CloseTerminal$json = {
  '1': 'CloseTerminal',
  '2': [
    {'1': 'reason', '3': 1, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `CloseTerminal`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List closeTerminalDescriptor = $convert
    .base64Decode('Cg1DbG9zZVRlcm1pbmFsEhYKBnJlYXNvbhgBIAEoCVIGcmVhc29u');

// This is a generated file - do not edit.
//
// Generated from coderoam/control/v1/session.proto.

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

@$core.Deprecated('Use resumeRequestDescriptor instead')
const ResumeRequest$json = {
  '1': 'ResumeRequest',
  '2': [
    {
      '1': 'previous_session_id',
      '3': 1,
      '4': 1,
      '5': 12,
      '10': 'previousSessionId'
    },
    {
      '1': 'acknowledged_sequences',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.coderoam.control.v1.ResumeRequest.AcknowledgedSequencesEntry',
      '10': 'acknowledgedSequences'
    },
  ],
  '3': [ResumeRequest_AcknowledgedSequencesEntry$json],
};

@$core.Deprecated('Use resumeRequestDescriptor instead')
const ResumeRequest_AcknowledgedSequencesEntry$json = {
  '1': 'AcknowledgedSequencesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 13, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 4, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `ResumeRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resumeRequestDescriptor = $convert.base64Decode(
    'Cg1SZXN1bWVSZXF1ZXN0Ei4KE3ByZXZpb3VzX3Nlc3Npb25faWQYASABKAxSEXByZXZpb3VzU2'
    'Vzc2lvbklkEnQKFmFja25vd2xlZGdlZF9zZXF1ZW5jZXMYAiADKAsyPS5jb2Rlcm9hbS5jb250'
    'cm9sLnYxLlJlc3VtZVJlcXVlc3QuQWNrbm93bGVkZ2VkU2VxdWVuY2VzRW50cnlSFWFja25vd2'
    'xlZGdlZFNlcXVlbmNlcxpIChpBY2tub3dsZWRnZWRTZXF1ZW5jZXNFbnRyeRIQCgNrZXkYASAB'
    'KA1SA2tleRIUCgV2YWx1ZRgCIAEoBFIFdmFsdWU6AjgB');

@$core.Deprecated('Use resumeAcceptedDescriptor instead')
const ResumeAccepted$json = {
  '1': 'ResumeAccepted',
  '2': [
    {'1': 'session_id', '3': 1, '4': 1, '5': 12, '10': 'sessionId'},
    {
      '1': 'replay_starts',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.coderoam.control.v1.ResumeAccepted.ReplayStartsEntry',
      '10': 'replayStarts'
    },
  ],
  '3': [ResumeAccepted_ReplayStartsEntry$json],
};

@$core.Deprecated('Use resumeAcceptedDescriptor instead')
const ResumeAccepted_ReplayStartsEntry$json = {
  '1': 'ReplayStartsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 13, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 4, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `ResumeAccepted`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resumeAcceptedDescriptor = $convert.base64Decode(
    'Cg5SZXN1bWVBY2NlcHRlZBIdCgpzZXNzaW9uX2lkGAEgASgMUglzZXNzaW9uSWQSWgoNcmVwbG'
    'F5X3N0YXJ0cxgCIAMoCzI1LmNvZGVyb2FtLmNvbnRyb2wudjEuUmVzdW1lQWNjZXB0ZWQuUmVw'
    'bGF5U3RhcnRzRW50cnlSDHJlcGxheVN0YXJ0cxo/ChFSZXBsYXlTdGFydHNFbnRyeRIQCgNrZX'
    'kYASABKA1SA2tleRIUCgV2YWx1ZRgCIAEoBFIFdmFsdWU6AjgB');

@$core.Deprecated('Use resumeUnavailableDescriptor instead')
const ResumeUnavailable$json = {
  '1': 'ResumeUnavailable',
  '2': [
    {'1': 'reason', '3': 1, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `ResumeUnavailable`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resumeUnavailableDescriptor = $convert.base64Decode(
    'ChFSZXN1bWVVbmF2YWlsYWJsZRIWCgZyZWFzb24YASABKAlSBnJlYXNvbg==');

@$core.Deprecated('Use heartbeatDescriptor instead')
const Heartbeat$json = {
  '1': 'Heartbeat',
  '2': [
    {
      '1': 'sent_at_unix_millis',
      '3': 1,
      '4': 1,
      '5': 3,
      '10': 'sentAtUnixMillis'
    },
  ],
};

/// Descriptor for `Heartbeat`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List heartbeatDescriptor = $convert.base64Decode(
    'CglIZWFydGJlYXQSLQoTc2VudF9hdF91bml4X21pbGxpcxgBIAEoA1IQc2VudEF0VW5peE1pbG'
    'xpcw==');

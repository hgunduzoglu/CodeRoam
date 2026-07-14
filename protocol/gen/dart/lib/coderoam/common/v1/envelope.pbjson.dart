// This is a generated file - do not edit.
//
// Generated from coderoam/common/v1/envelope.proto.

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

@$core.Deprecated('Use channelKindDescriptor instead')
const ChannelKind$json = {
  '1': 'ChannelKind',
  '2': [
    {'1': 'CHANNEL_KIND_UNSPECIFIED', '2': 0},
    {'1': 'CHANNEL_KIND_CONTROL', '2': 1},
    {'1': 'CHANNEL_KIND_FILESYSTEM', '2': 2},
    {'1': 'CHANNEL_KIND_EDITOR', '2': 3},
    {'1': 'CHANNEL_KIND_TERMINAL', '2': 4},
    {'1': 'CHANNEL_KIND_GIT', '2': 5},
    {'1': 'CHANNEL_KIND_LSP', '2': 6},
    {'1': 'CHANNEL_KIND_AI', '2': 7},
    {'1': 'CHANNEL_KIND_LOGS', '2': 8},
    {'1': 'CHANNEL_KIND_PORT_FORWARD', '2': 9},
  ],
};

/// Descriptor for `ChannelKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List channelKindDescriptor = $convert.base64Decode(
    'CgtDaGFubmVsS2luZBIcChhDSEFOTkVMX0tJTkRfVU5TUEVDSUZJRUQQABIYChRDSEFOTkVMX0'
    'tJTkRfQ09OVFJPTBABEhsKF0NIQU5ORUxfS0lORF9GSUxFU1lTVEVNEAISFwoTQ0hBTk5FTF9L'
    'SU5EX0VESVRPUhADEhkKFUNIQU5ORUxfS0lORF9URVJNSU5BTBAEEhQKEENIQU5ORUxfS0lORF'
    '9HSVQQBRIUChBDSEFOTkVMX0tJTkRfTFNQEAYSEwoPQ0hBTk5FTF9LSU5EX0FJEAcSFQoRQ0hB'
    'Tk5FTF9LSU5EX0xPR1MQCBIdChlDSEFOTkVMX0tJTkRfUE9SVF9GT1JXQVJEEAk=');

@$core.Deprecated('Use encryptedFrameDescriptor instead')
const EncryptedFrame$json = {
  '1': 'EncryptedFrame',
  '2': [
    {'1': 'protocol_version', '3': 1, '4': 1, '5': 13, '10': 'protocolVersion'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 12, '10': 'sessionId'},
    {
      '1': 'channel',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.coderoam.common.v1.ChannelKind',
      '10': 'channel'
    },
    {'1': 'stream_id', '3': 4, '4': 1, '5': 13, '10': 'streamId'},
    {'1': 'sequence', '3': 5, '4': 1, '5': 4, '10': 'sequence'},
    {'1': 'acknowledgement', '3': 6, '4': 1, '5': 4, '10': 'acknowledgement'},
    {'1': 'flags', '3': 7, '4': 1, '5': 13, '10': 'flags'},
    {'1': 'ciphertext', '3': 8, '4': 1, '5': 12, '10': 'ciphertext'},
  ],
};

/// Descriptor for `EncryptedFrame`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List encryptedFrameDescriptor = $convert.base64Decode(
    'Cg5FbmNyeXB0ZWRGcmFtZRIpChBwcm90b2NvbF92ZXJzaW9uGAEgASgNUg9wcm90b2NvbFZlcn'
    'Npb24SHQoKc2Vzc2lvbl9pZBgCIAEoDFIJc2Vzc2lvbklkEjkKB2NoYW5uZWwYAyABKA4yHy5j'
    'b2Rlcm9hbS5jb21tb24udjEuQ2hhbm5lbEtpbmRSB2NoYW5uZWwSGwoJc3RyZWFtX2lkGAQgAS'
    'gNUghzdHJlYW1JZBIaCghzZXF1ZW5jZRgFIAEoBFIIc2VxdWVuY2USKAoPYWNrbm93bGVkZ2Vt'
    'ZW50GAYgASgEUg9hY2tub3dsZWRnZW1lbnQSFAoFZmxhZ3MYByABKA1SBWZsYWdzEh4KCmNpcG'
    'hlcnRleHQYCCABKAxSCmNpcGhlcnRleHQ=');

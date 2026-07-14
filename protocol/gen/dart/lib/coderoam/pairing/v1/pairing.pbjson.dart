// This is a generated file - do not edit.
//
// Generated from coderoam/pairing/v1/pairing.proto.

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

@$core.Deprecated('Use pairingQrPayloadDescriptor instead')
const PairingQrPayload$json = {
  '1': 'PairingQrPayload',
  '2': [
    {'1': 'pairing_id', '3': 1, '4': 1, '5': 9, '10': 'pairingId'},
    {
      '1': 'agent_static_public_key',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'agentStaticPublicKey'
    },
    {
      '1': 'agent_key_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'agentKeyFingerprint'
    },
    {'1': 'pairing_secret', '3': 4, '4': 1, '5': 12, '10': 'pairingSecret'},
    {'1': 'protocol_version', '3': 5, '4': 1, '5': 13, '10': 'protocolVersion'},
    {
      '1': 'expires_at_unix_seconds',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixSeconds'
    },
  ],
};

/// Descriptor for `PairingQrPayload`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingQrPayloadDescriptor = $convert.base64Decode(
    'ChBQYWlyaW5nUXJQYXlsb2FkEh0KCnBhaXJpbmdfaWQYASABKAlSCXBhaXJpbmdJZBI1ChdhZ2'
    'VudF9zdGF0aWNfcHVibGljX2tleRgCIAEoDFIUYWdlbnRTdGF0aWNQdWJsaWNLZXkSMgoVYWdl'
    'bnRfa2V5X2ZpbmdlcnByaW50GAMgASgJUhNhZ2VudEtleUZpbmdlcnByaW50EiUKDnBhaXJpbm'
    'dfc2VjcmV0GAQgASgMUg1wYWlyaW5nU2VjcmV0EikKEHByb3RvY29sX3ZlcnNpb24YBSABKA1S'
    'D3Byb3RvY29sVmVyc2lvbhI1ChdleHBpcmVzX2F0X3VuaXhfc2Vjb25kcxgGIAEoA1IUZXhwaX'
    'Jlc0F0VW5peFNlY29uZHM=');

@$core.Deprecated('Use pairingCompleteDescriptor instead')
const PairingComplete$json = {
  '1': 'PairingComplete',
  '2': [
    {'1': 'pairing_id', '3': 1, '4': 1, '5': 9, '10': 'pairingId'},
    {
      '1': 'device_static_public_key',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'deviceStaticPublicKey'
    },
    {
      '1': 'device_key_fingerprint',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'deviceKeyFingerprint'
    },
    {
      '1': 'agent_static_public_key',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'agentStaticPublicKey'
    },
    {
      '1': 'agent_key_fingerprint',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'agentKeyFingerprint'
    },
  ],
};

/// Descriptor for `PairingComplete`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingCompleteDescriptor = $convert.base64Decode(
    'Cg9QYWlyaW5nQ29tcGxldGUSHQoKcGFpcmluZ19pZBgBIAEoCVIJcGFpcmluZ0lkEjcKGGRldm'
    'ljZV9zdGF0aWNfcHVibGljX2tleRgCIAEoDFIVZGV2aWNlU3RhdGljUHVibGljS2V5EjQKFmRl'
    'dmljZV9rZXlfZmluZ2VycHJpbnQYAyABKAlSFGRldmljZUtleUZpbmdlcnByaW50EjUKF2FnZW'
    '50X3N0YXRpY19wdWJsaWNfa2V5GAQgASgMUhRhZ2VudFN0YXRpY1B1YmxpY0tleRIyChVhZ2Vu'
    'dF9rZXlfZmluZ2VycHJpbnQYBSABKAlSE2FnZW50S2V5RmluZ2VycHJpbnQ=');

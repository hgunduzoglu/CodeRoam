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

@$core.Deprecated('Use pairingEndpointRoleDescriptor instead')
const PairingEndpointRole$json = {
  '1': 'PairingEndpointRole',
  '2': [
    {'1': 'PAIRING_ENDPOINT_ROLE_UNSPECIFIED', '2': 0},
    {'1': 'PAIRING_ENDPOINT_ROLE_MOBILE', '2': 1},
    {'1': 'PAIRING_ENDPOINT_ROLE_AGENT', '2': 2},
  ],
};

/// Descriptor for `PairingEndpointRole`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List pairingEndpointRoleDescriptor = $convert.base64Decode(
    'ChNQYWlyaW5nRW5kcG9pbnRSb2xlEiUKIVBBSVJJTkdfRU5EUE9JTlRfUk9MRV9VTlNQRUNJRk'
    'lFRBAAEiAKHFBBSVJJTkdfRU5EUE9JTlRfUk9MRV9NT0JJTEUQARIfChtQQUlSSU5HX0VORFBP'
    'SU5UX1JPTEVfQUdFTlQQAg==');

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
    {
      '1': 'agent_display_name',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'agentDisplayName'
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
    'Jlc0F0VW5peFNlY29uZHMSLAoSYWdlbnRfZGlzcGxheV9uYW1lGAcgASgJUhBhZ2VudERpc3Bs'
    'YXlOYW1l');

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

@$core.Deprecated('Use pairingHandshakePayloadDescriptor instead')
const PairingHandshakePayload$json = {
  '1': 'PairingHandshakePayload',
  '2': [
    {'1': 'protocol_version', '3': 1, '4': 1, '5': 13, '10': 'protocolVersion'},
    {'1': 'pairing_id', '3': 2, '4': 1, '5': 9, '10': 'pairingId'},
    {
      '1': 'role',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.coderoam.pairing.v1.PairingEndpointRole',
      '10': 'role'
    },
    {
      '1': 'static_public_key',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'staticPublicKey'
    },
    {'1': 'key_fingerprint', '3': 5, '4': 1, '5': 9, '10': 'keyFingerprint'},
  ],
};

/// Descriptor for `PairingHandshakePayload`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingHandshakePayloadDescriptor = $convert.base64Decode(
    'ChdQYWlyaW5nSGFuZHNoYWtlUGF5bG9hZBIpChBwcm90b2NvbF92ZXJzaW9uGAEgASgNUg9wcm'
    '90b2NvbFZlcnNpb24SHQoKcGFpcmluZ19pZBgCIAEoCVIJcGFpcmluZ0lkEjwKBHJvbGUYAyAB'
    'KA4yKC5jb2Rlcm9hbS5wYWlyaW5nLnYxLlBhaXJpbmdFbmRwb2ludFJvbGVSBHJvbGUSKgoRc3'
    'RhdGljX3B1YmxpY19rZXkYBCABKAxSD3N0YXRpY1B1YmxpY0tleRInCg9rZXlfZmluZ2VycHJp'
    'bnQYBSABKAlSDmtleUZpbmdlcnByaW50');

@$core.Deprecated('Use pairingConfirmationDescriptor instead')
const PairingConfirmation$json = {
  '1': 'PairingConfirmation',
  '2': [
    {'1': 'pairing_id', '3': 1, '4': 1, '5': 9, '10': 'pairingId'},
    {'1': 'protocol_version', '3': 2, '4': 1, '5': 13, '10': 'protocolVersion'},
    {
      '1': 'role',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.coderoam.pairing.v1.PairingEndpointRole',
      '10': 'role'
    },
    {'1': 'channel_binding', '3': 4, '4': 1, '5': 12, '10': 'channelBinding'},
    {
      '1': 'observed_peer_static_public_key',
      '3': 5,
      '4': 1,
      '5': 12,
      '10': 'observedPeerStaticPublicKey'
    },
    {
      '1': 'observed_peer_key_fingerprint',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'observedPeerKeyFingerprint'
    },
  ],
};

/// Descriptor for `PairingConfirmation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pairingConfirmationDescriptor = $convert.base64Decode(
    'ChNQYWlyaW5nQ29uZmlybWF0aW9uEh0KCnBhaXJpbmdfaWQYASABKAlSCXBhaXJpbmdJZBIpCh'
    'Bwcm90b2NvbF92ZXJzaW9uGAIgASgNUg9wcm90b2NvbFZlcnNpb24SPAoEcm9sZRgDIAEoDjIo'
    'LmNvZGVyb2FtLnBhaXJpbmcudjEuUGFpcmluZ0VuZHBvaW50Um9sZVIEcm9sZRInCg9jaGFubm'
    'VsX2JpbmRpbmcYBCABKAxSDmNoYW5uZWxCaW5kaW5nEkQKH29ic2VydmVkX3BlZXJfc3RhdGlj'
    'X3B1YmxpY19rZXkYBSABKAxSG29ic2VydmVkUGVlclN0YXRpY1B1YmxpY0tleRJBCh1vYnNlcn'
    'ZlZF9wZWVyX2tleV9maW5nZXJwcmludBgGIAEoCVIab2JzZXJ2ZWRQZWVyS2V5RmluZ2VycHJp'
    'bnQ=');

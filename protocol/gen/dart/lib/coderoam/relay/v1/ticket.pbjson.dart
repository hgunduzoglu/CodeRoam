// This is a generated file - do not edit.
//
// Generated from coderoam/relay/v1/ticket.proto.

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

@$core.Deprecated('Use endpointRoleDescriptor instead')
const EndpointRole$json = {
  '1': 'EndpointRole',
  '2': [
    {'1': 'ENDPOINT_ROLE_UNSPECIFIED', '2': 0},
    {'1': 'ENDPOINT_ROLE_CLIENT', '2': 1},
    {'1': 'ENDPOINT_ROLE_AGENT', '2': 2},
  ],
};

/// Descriptor for `EndpointRole`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List endpointRoleDescriptor = $convert.base64Decode(
    'CgxFbmRwb2ludFJvbGUSHQoZRU5EUE9JTlRfUk9MRV9VTlNQRUNJRklFRBAAEhgKFEVORFBPSU'
    '5UX1JPTEVfQ0xJRU5UEAESFwoTRU5EUE9JTlRfUk9MRV9BR0VOVBAC');

@$core.Deprecated('Use ticketPurposeDescriptor instead')
const TicketPurpose$json = {
  '1': 'TicketPurpose',
  '2': [
    {'1': 'TICKET_PURPOSE_UNSPECIFIED', '2': 0},
    {'1': 'TICKET_PURPOSE_PAIRING', '2': 1},
    {'1': 'TICKET_PURPOSE_SESSION', '2': 2},
  ],
};

/// Descriptor for `TicketPurpose`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List ticketPurposeDescriptor = $convert.base64Decode(
    'Cg1UaWNrZXRQdXJwb3NlEh4KGlRJQ0tFVF9QVVJQT1NFX1VOU1BFQ0lGSUVEEAASGgoWVElDS0'
    'VUX1BVUlBPU0VfUEFJUklORxABEhoKFlRJQ0tFVF9QVVJQT1NFX1NFU1NJT04QAg==');

@$core.Deprecated('Use connectionTicketClaimsDescriptor instead')
const ConnectionTicketClaims$json = {
  '1': 'ConnectionTicketClaims',
  '2': [
    {'1': 'ticket_id', '3': 1, '4': 1, '5': 9, '10': 'ticketId'},
    {'1': 'route_id', '3': 2, '4': 1, '5': 9, '10': 'routeId'},
    {
      '1': 'role',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.coderoam.relay.v1.EndpointRole',
      '10': 'role'
    },
    {'1': 'endpoint_id', '3': 4, '4': 1, '5': 9, '10': 'endpointId'},
    {'1': 'relay_region', '3': 5, '4': 1, '5': 9, '10': 'relayRegion'},
    {
      '1': 'issued_at_unix_seconds',
      '3': 6,
      '4': 1,
      '5': 3,
      '10': 'issuedAtUnixSeconds'
    },
    {
      '1': 'expires_at_unix_seconds',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'expiresAtUnixSeconds'
    },
    {'1': 'nonce', '3': 8, '4': 1, '5': 12, '10': 'nonce'},
    {
      '1': 'purpose',
      '3': 9,
      '4': 1,
      '5': 14,
      '6': '.coderoam.relay.v1.TicketPurpose',
      '10': 'purpose'
    },
    {
      '1': 'protocol_version',
      '3': 10,
      '4': 1,
      '5': 13,
      '10': 'protocolVersion'
    },
    {
      '1': 'not_before_unix_seconds',
      '3': 11,
      '4': 1,
      '5': 3,
      '10': 'notBeforeUnixSeconds'
    },
    {'1': 'key_id', '3': 12, '4': 1, '5': 9, '10': 'keyId'},
  ],
};

/// Descriptor for `ConnectionTicketClaims`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionTicketClaimsDescriptor = $convert.base64Decode(
    'ChZDb25uZWN0aW9uVGlja2V0Q2xhaW1zEhsKCXRpY2tldF9pZBgBIAEoCVIIdGlja2V0SWQSGQ'
    'oIcm91dGVfaWQYAiABKAlSB3JvdXRlSWQSMwoEcm9sZRgDIAEoDjIfLmNvZGVyb2FtLnJlbGF5'
    'LnYxLkVuZHBvaW50Um9sZVIEcm9sZRIfCgtlbmRwb2ludF9pZBgEIAEoCVIKZW5kcG9pbnRJZB'
    'IhCgxyZWxheV9yZWdpb24YBSABKAlSC3JlbGF5UmVnaW9uEjMKFmlzc3VlZF9hdF91bml4X3Nl'
    'Y29uZHMYBiABKANSE2lzc3VlZEF0VW5peFNlY29uZHMSNQoXZXhwaXJlc19hdF91bml4X3NlY2'
    '9uZHMYByABKANSFGV4cGlyZXNBdFVuaXhTZWNvbmRzEhQKBW5vbmNlGAggASgMUgVub25jZRI6'
    'CgdwdXJwb3NlGAkgASgOMiAuY29kZXJvYW0ucmVsYXkudjEuVGlja2V0UHVycG9zZVIHcHVycG'
    '9zZRIpChBwcm90b2NvbF92ZXJzaW9uGAogASgNUg9wcm90b2NvbFZlcnNpb24SNQoXbm90X2Jl'
    'Zm9yZV91bml4X3NlY29uZHMYCyABKANSFG5vdEJlZm9yZVVuaXhTZWNvbmRzEhUKBmtleV9pZB'
    'gMIAEoCVIFa2V5SWQ=');

@$core.Deprecated('Use signedConnectionTicketDescriptor instead')
const SignedConnectionTicket$json = {
  '1': 'SignedConnectionTicket',
  '2': [
    {'1': 'claims', '3': 1, '4': 1, '5': 12, '10': 'claims'},
    {'1': 'signature', '3': 2, '4': 1, '5': 12, '10': 'signature'},
  ],
};

/// Descriptor for `SignedConnectionTicket`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signedConnectionTicketDescriptor =
    $convert.base64Decode(
        'ChZTaWduZWRDb25uZWN0aW9uVGlja2V0EhYKBmNsYWltcxgBIAEoDFIGY2xhaW1zEhwKCXNpZ2'
        '5hdHVyZRgCIAEoDFIJc2lnbmF0dXJl');

// This is a generated file - do not edit.
//
// Generated from coderoam/pairing/v1/pairing.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class PairingEndpointRole extends $pb.ProtobufEnum {
  static const PairingEndpointRole PAIRING_ENDPOINT_ROLE_UNSPECIFIED =
      PairingEndpointRole._(
          0, _omitEnumNames ? '' : 'PAIRING_ENDPOINT_ROLE_UNSPECIFIED');
  static const PairingEndpointRole PAIRING_ENDPOINT_ROLE_MOBILE =
      PairingEndpointRole._(
          1, _omitEnumNames ? '' : 'PAIRING_ENDPOINT_ROLE_MOBILE');
  static const PairingEndpointRole PAIRING_ENDPOINT_ROLE_AGENT =
      PairingEndpointRole._(
          2, _omitEnumNames ? '' : 'PAIRING_ENDPOINT_ROLE_AGENT');

  static const $core.List<PairingEndpointRole> values = <PairingEndpointRole>[
    PAIRING_ENDPOINT_ROLE_UNSPECIFIED,
    PAIRING_ENDPOINT_ROLE_MOBILE,
    PAIRING_ENDPOINT_ROLE_AGENT,
  ];

  static final $core.List<PairingEndpointRole?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static PairingEndpointRole? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PairingEndpointRole._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');

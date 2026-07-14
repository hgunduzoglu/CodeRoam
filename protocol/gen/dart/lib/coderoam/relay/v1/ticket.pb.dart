// This is a generated file - do not edit.
//
// Generated from coderoam/relay/v1/ticket.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'ticket.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'ticket.pbenum.dart';

class ConnectionTicketClaims extends $pb.GeneratedMessage {
  factory ConnectionTicketClaims({
    $core.String? ticketId,
    $core.String? routeId,
    EndpointRole? role,
    $core.String? endpointId,
    $core.String? relayRegion,
    $fixnum.Int64? issuedAtUnixSeconds,
    $fixnum.Int64? expiresAtUnixSeconds,
    $core.List<$core.int>? nonce,
  }) {
    final result = create();
    if (ticketId != null) result.ticketId = ticketId;
    if (routeId != null) result.routeId = routeId;
    if (role != null) result.role = role;
    if (endpointId != null) result.endpointId = endpointId;
    if (relayRegion != null) result.relayRegion = relayRegion;
    if (issuedAtUnixSeconds != null)
      result.issuedAtUnixSeconds = issuedAtUnixSeconds;
    if (expiresAtUnixSeconds != null)
      result.expiresAtUnixSeconds = expiresAtUnixSeconds;
    if (nonce != null) result.nonce = nonce;
    return result;
  }

  ConnectionTicketClaims._();

  factory ConnectionTicketClaims.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionTicketClaims.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionTicketClaims',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.relay.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'ticketId')
    ..aOS(2, _omitFieldNames ? '' : 'routeId')
    ..aE<EndpointRole>(3, _omitFieldNames ? '' : 'role',
        enumValues: EndpointRole.values)
    ..aOS(4, _omitFieldNames ? '' : 'endpointId')
    ..aOS(5, _omitFieldNames ? '' : 'relayRegion')
    ..aInt64(6, _omitFieldNames ? '' : 'issuedAtUnixSeconds')
    ..aInt64(7, _omitFieldNames ? '' : 'expiresAtUnixSeconds')
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'nonce', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionTicketClaims clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionTicketClaims copyWith(
          void Function(ConnectionTicketClaims) updates) =>
      super.copyWith((message) => updates(message as ConnectionTicketClaims))
          as ConnectionTicketClaims;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionTicketClaims create() => ConnectionTicketClaims._();
  @$core.override
  ConnectionTicketClaims createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionTicketClaims getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionTicketClaims>(create);
  static ConnectionTicketClaims? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get ticketId => $_getSZ(0);
  @$pb.TagNumber(1)
  set ticketId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTicketId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTicketId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get routeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set routeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRouteId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRouteId() => $_clearField(2);

  @$pb.TagNumber(3)
  EndpointRole get role => $_getN(2);
  @$pb.TagNumber(3)
  set role(EndpointRole value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRole() => $_has(2);
  @$pb.TagNumber(3)
  void clearRole() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get endpointId => $_getSZ(3);
  @$pb.TagNumber(4)
  set endpointId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEndpointId() => $_has(3);
  @$pb.TagNumber(4)
  void clearEndpointId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get relayRegion => $_getSZ(4);
  @$pb.TagNumber(5)
  set relayRegion($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRelayRegion() => $_has(4);
  @$pb.TagNumber(5)
  void clearRelayRegion() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get issuedAtUnixSeconds => $_getI64(5);
  @$pb.TagNumber(6)
  set issuedAtUnixSeconds($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIssuedAtUnixSeconds() => $_has(5);
  @$pb.TagNumber(6)
  void clearIssuedAtUnixSeconds() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get expiresAtUnixSeconds => $_getI64(6);
  @$pb.TagNumber(7)
  set expiresAtUnixSeconds($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasExpiresAtUnixSeconds() => $_has(6);
  @$pb.TagNumber(7)
  void clearExpiresAtUnixSeconds() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.List<$core.int> get nonce => $_getN(7);
  @$pb.TagNumber(8)
  set nonce($core.List<$core.int> value) => $_setBytes(7, value);
  @$pb.TagNumber(8)
  $core.bool hasNonce() => $_has(7);
  @$pb.TagNumber(8)
  void clearNonce() => $_clearField(8);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');

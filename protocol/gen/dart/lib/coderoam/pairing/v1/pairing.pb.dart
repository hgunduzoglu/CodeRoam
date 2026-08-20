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

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'pairing.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'pairing.pbenum.dart';

class PairingQrPayload extends $pb.GeneratedMessage {
  factory PairingQrPayload({
    $core.String? pairingId,
    $core.List<$core.int>? agentStaticPublicKey,
    $core.String? agentKeyFingerprint,
    $core.List<$core.int>? pairingSecret,
    $core.int? protocolVersion,
    $fixnum.Int64? expiresAtUnixSeconds,
    $core.String? agentDisplayName,
  }) {
    final result = create();
    if (pairingId != null) result.pairingId = pairingId;
    if (agentStaticPublicKey != null)
      result.agentStaticPublicKey = agentStaticPublicKey;
    if (agentKeyFingerprint != null)
      result.agentKeyFingerprint = agentKeyFingerprint;
    if (pairingSecret != null) result.pairingSecret = pairingSecret;
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (expiresAtUnixSeconds != null)
      result.expiresAtUnixSeconds = expiresAtUnixSeconds;
    if (agentDisplayName != null) result.agentDisplayName = agentDisplayName;
    return result;
  }

  PairingQrPayload._();

  factory PairingQrPayload.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingQrPayload.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingQrPayload',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.pairing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'pairingId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'agentStaticPublicKey', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'agentKeyFingerprint')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'pairingSecret', $pb.PbFieldType.OY)
    ..aI(5, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aInt64(6, _omitFieldNames ? '' : 'expiresAtUnixSeconds')
    ..aOS(7, _omitFieldNames ? '' : 'agentDisplayName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingQrPayload clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingQrPayload copyWith(void Function(PairingQrPayload) updates) =>
      super.copyWith((message) => updates(message as PairingQrPayload))
          as PairingQrPayload;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingQrPayload create() => PairingQrPayload._();
  @$core.override
  PairingQrPayload createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingQrPayload getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingQrPayload>(create);
  static PairingQrPayload? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get pairingId => $_getSZ(0);
  @$pb.TagNumber(1)
  set pairingId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPairingId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPairingId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get agentStaticPublicKey => $_getN(1);
  @$pb.TagNumber(2)
  set agentStaticPublicKey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAgentStaticPublicKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearAgentStaticPublicKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get agentKeyFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set agentKeyFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAgentKeyFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearAgentKeyFingerprint() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get pairingSecret => $_getN(3);
  @$pb.TagNumber(4)
  set pairingSecret($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPairingSecret() => $_has(3);
  @$pb.TagNumber(4)
  void clearPairingSecret() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get protocolVersion => $_getIZ(4);
  @$pb.TagNumber(5)
  set protocolVersion($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProtocolVersion() => $_has(4);
  @$pb.TagNumber(5)
  void clearProtocolVersion() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get expiresAtUnixSeconds => $_getI64(5);
  @$pb.TagNumber(6)
  set expiresAtUnixSeconds($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasExpiresAtUnixSeconds() => $_has(5);
  @$pb.TagNumber(6)
  void clearExpiresAtUnixSeconds() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get agentDisplayName => $_getSZ(6);
  @$pb.TagNumber(7)
  set agentDisplayName($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasAgentDisplayName() => $_has(6);
  @$pb.TagNumber(7)
  void clearAgentDisplayName() => $_clearField(7);
}

class PairingComplete extends $pb.GeneratedMessage {
  factory PairingComplete({
    $core.String? pairingId,
    $core.List<$core.int>? deviceStaticPublicKey,
    $core.String? deviceKeyFingerprint,
    $core.List<$core.int>? agentStaticPublicKey,
    $core.String? agentKeyFingerprint,
  }) {
    final result = create();
    if (pairingId != null) result.pairingId = pairingId;
    if (deviceStaticPublicKey != null)
      result.deviceStaticPublicKey = deviceStaticPublicKey;
    if (deviceKeyFingerprint != null)
      result.deviceKeyFingerprint = deviceKeyFingerprint;
    if (agentStaticPublicKey != null)
      result.agentStaticPublicKey = agentStaticPublicKey;
    if (agentKeyFingerprint != null)
      result.agentKeyFingerprint = agentKeyFingerprint;
    return result;
  }

  PairingComplete._();

  factory PairingComplete.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingComplete.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingComplete',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.pairing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'pairingId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'deviceStaticPublicKey', $pb.PbFieldType.OY)
    ..aOS(3, _omitFieldNames ? '' : 'deviceKeyFingerprint')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'agentStaticPublicKey', $pb.PbFieldType.OY)
    ..aOS(5, _omitFieldNames ? '' : 'agentKeyFingerprint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingComplete clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingComplete copyWith(void Function(PairingComplete) updates) =>
      super.copyWith((message) => updates(message as PairingComplete))
          as PairingComplete;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingComplete create() => PairingComplete._();
  @$core.override
  PairingComplete createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingComplete getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingComplete>(create);
  static PairingComplete? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get pairingId => $_getSZ(0);
  @$pb.TagNumber(1)
  set pairingId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPairingId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPairingId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get deviceStaticPublicKey => $_getN(1);
  @$pb.TagNumber(2)
  set deviceStaticPublicKey($core.List<$core.int> value) =>
      $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceStaticPublicKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceStaticPublicKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceKeyFingerprint => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceKeyFingerprint($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceKeyFingerprint() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceKeyFingerprint() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get agentStaticPublicKey => $_getN(3);
  @$pb.TagNumber(4)
  set agentStaticPublicKey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAgentStaticPublicKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearAgentStaticPublicKey() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get agentKeyFingerprint => $_getSZ(4);
  @$pb.TagNumber(5)
  set agentKeyFingerprint($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAgentKeyFingerprint() => $_has(4);
  @$pb.TagNumber(5)
  void clearAgentKeyFingerprint() => $_clearField(5);
}

/// PairingHandshakePayload is carried only inside the authenticated Noise
/// handshake. Consumers must recompute the fingerprint and compare the static
/// key with the key recovered from Noise before trusting either field.
class PairingHandshakePayload extends $pb.GeneratedMessage {
  factory PairingHandshakePayload({
    $core.int? protocolVersion,
    $core.String? pairingId,
    PairingEndpointRole? role,
    $core.List<$core.int>? staticPublicKey,
    $core.String? keyFingerprint,
  }) {
    final result = create();
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (pairingId != null) result.pairingId = pairingId;
    if (role != null) result.role = role;
    if (staticPublicKey != null) result.staticPublicKey = staticPublicKey;
    if (keyFingerprint != null) result.keyFingerprint = keyFingerprint;
    return result;
  }

  PairingHandshakePayload._();

  factory PairingHandshakePayload.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingHandshakePayload.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingHandshakePayload',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.pairing.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'pairingId')
    ..aE<PairingEndpointRole>(3, _omitFieldNames ? '' : 'role',
        enumValues: PairingEndpointRole.values)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'staticPublicKey', $pb.PbFieldType.OY)
    ..aOS(5, _omitFieldNames ? '' : 'keyFingerprint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingHandshakePayload clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingHandshakePayload copyWith(
          void Function(PairingHandshakePayload) updates) =>
      super.copyWith((message) => updates(message as PairingHandshakePayload))
          as PairingHandshakePayload;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingHandshakePayload create() => PairingHandshakePayload._();
  @$core.override
  PairingHandshakePayload createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingHandshakePayload getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingHandshakePayload>(create);
  static PairingHandshakePayload? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get protocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set protocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get pairingId => $_getSZ(1);
  @$pb.TagNumber(2)
  set pairingId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPairingId() => $_has(1);
  @$pb.TagNumber(2)
  void clearPairingId() => $_clearField(2);

  @$pb.TagNumber(3)
  PairingEndpointRole get role => $_getN(2);
  @$pb.TagNumber(3)
  set role(PairingEndpointRole value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRole() => $_has(2);
  @$pb.TagNumber(3)
  void clearRole() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get staticPublicKey => $_getN(3);
  @$pb.TagNumber(4)
  set staticPublicKey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStaticPublicKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearStaticPublicKey() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get keyFingerprint => $_getSZ(4);
  @$pb.TagNumber(5)
  set keyFingerprint($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasKeyFingerprint() => $_has(4);
  @$pb.TagNumber(5)
  void clearKeyFingerprint() => $_clearField(5);
}

/// PairingConfirmation binds one endpoint's observation of the completed
/// handshake to the pairing attempt. The application layer requires a
/// 32-byte channel binding and an exact peer-key match.
class PairingConfirmation extends $pb.GeneratedMessage {
  factory PairingConfirmation({
    $core.String? pairingId,
    $core.int? protocolVersion,
    PairingEndpointRole? role,
    $core.List<$core.int>? channelBinding,
    $core.List<$core.int>? observedPeerStaticPublicKey,
    $core.String? observedPeerKeyFingerprint,
  }) {
    final result = create();
    if (pairingId != null) result.pairingId = pairingId;
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (role != null) result.role = role;
    if (channelBinding != null) result.channelBinding = channelBinding;
    if (observedPeerStaticPublicKey != null)
      result.observedPeerStaticPublicKey = observedPeerStaticPublicKey;
    if (observedPeerKeyFingerprint != null)
      result.observedPeerKeyFingerprint = observedPeerKeyFingerprint;
    return result;
  }

  PairingConfirmation._();

  factory PairingConfirmation.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PairingConfirmation.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PairingConfirmation',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'coderoam.pairing.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'pairingId')
    ..aI(2, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aE<PairingEndpointRole>(3, _omitFieldNames ? '' : 'role',
        enumValues: PairingEndpointRole.values)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'channelBinding', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5,
        _omitFieldNames ? '' : 'observedPeerStaticPublicKey',
        $pb.PbFieldType.OY)
    ..aOS(6, _omitFieldNames ? '' : 'observedPeerKeyFingerprint')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingConfirmation clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PairingConfirmation copyWith(void Function(PairingConfirmation) updates) =>
      super.copyWith((message) => updates(message as PairingConfirmation))
          as PairingConfirmation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PairingConfirmation create() => PairingConfirmation._();
  @$core.override
  PairingConfirmation createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PairingConfirmation getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PairingConfirmation>(create);
  static PairingConfirmation? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get pairingId => $_getSZ(0);
  @$pb.TagNumber(1)
  set pairingId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPairingId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPairingId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get protocolVersion => $_getIZ(1);
  @$pb.TagNumber(2)
  set protocolVersion($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasProtocolVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearProtocolVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  PairingEndpointRole get role => $_getN(2);
  @$pb.TagNumber(3)
  set role(PairingEndpointRole value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRole() => $_has(2);
  @$pb.TagNumber(3)
  void clearRole() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get channelBinding => $_getN(3);
  @$pb.TagNumber(4)
  set channelBinding($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasChannelBinding() => $_has(3);
  @$pb.TagNumber(4)
  void clearChannelBinding() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get observedPeerStaticPublicKey => $_getN(4);
  @$pb.TagNumber(5)
  set observedPeerStaticPublicKey($core.List<$core.int> value) =>
      $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasObservedPeerStaticPublicKey() => $_has(4);
  @$pb.TagNumber(5)
  void clearObservedPeerStaticPublicKey() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get observedPeerKeyFingerprint => $_getSZ(5);
  @$pb.TagNumber(6)
  set observedPeerKeyFingerprint($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasObservedPeerKeyFingerprint() => $_has(5);
  @$pb.TagNumber(6)
  void clearObservedPeerKeyFingerprint() => $_clearField(6);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');

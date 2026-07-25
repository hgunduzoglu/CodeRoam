package compat_test

import (
	"testing"

	pairingv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/pairing/v1"
	relayv1 "github.com/hgunduzoglu/coderoam/protocol/gen/go/coderoam/relay/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestLegacyTicketDecodesWithSafeM3Defaults(t *testing.T) {
	var legacy []byte
	legacy = protowire.AppendTag(legacy, 1, protowire.BytesType)
	legacy = protowire.AppendString(legacy, "ticket-legacy")
	legacy = protowire.AppendTag(legacy, 2, protowire.BytesType)
	legacy = protowire.AppendString(legacy, "route-legacy")
	legacy = protowire.AppendTag(legacy, 3, protowire.VarintType)
	legacy = protowire.AppendVarint(legacy, uint64(relayv1.EndpointRole_ENDPOINT_ROLE_CLIENT))
	legacy = protowire.AppendTag(legacy, 8, protowire.BytesType)
	legacy = protowire.AppendBytes(legacy, []byte("legacy-nonce"))

	var claims relayv1.ConnectionTicketClaims
	if err := proto.Unmarshal(legacy, &claims); err != nil {
		t.Fatalf("decode legacy ticket: %v", err)
	}
	if claims.GetTicketId() != "ticket-legacy" || claims.GetRouteId() != "route-legacy" {
		t.Fatalf(
			"legacy fields changed: ticket_id=%q route_id=%q",
			claims.GetTicketId(),
			claims.GetRouteId(),
		)
	}
	if claims.GetPurpose() != relayv1.TicketPurpose_TICKET_PURPOSE_UNSPECIFIED {
		t.Fatalf("legacy purpose must fail closed as unspecified: %v", claims.GetPurpose())
	}
	if claims.GetProtocolVersion() != 0 || claims.GetKeyId() != "" {
		t.Fatalf(
			"new ticket fields must retain safe zero values: version=%d key_id=%q",
			claims.GetProtocolVersion(),
			claims.GetKeyId(),
		)
	}
}

func TestM3PairingContractFieldNumbers(t *testing.T) {
	assertField(t, (&pairingv1.PairingQrPayload{}).ProtoReflect().Descriptor(), "agent_display_name", 7, protoreflect.StringKind)
	assertField(t, (&pairingv1.PairingHandshakePayload{}).ProtoReflect().Descriptor(), "static_public_key", 4, protoreflect.BytesKind)
	assertField(t, (&pairingv1.PairingConfirmation{}).ProtoReflect().Descriptor(), "channel_binding", 4, protoreflect.BytesKind)
	assertField(t, (&relayv1.ConnectionTicketClaims{}).ProtoReflect().Descriptor(), "purpose", 9, protoreflect.EnumKind)
	assertField(t, (&relayv1.ConnectionTicketClaims{}).ProtoReflect().Descriptor(), "key_id", 12, protoreflect.StringKind)
	assertField(t, (&relayv1.SignedConnectionTicket{}).ProtoReflect().Descriptor(), "signature", 2, protoreflect.BytesKind)
}

func TestSignedTicketRejectsMalformedWire(t *testing.T) {
	var ticket relayv1.SignedConnectionTicket
	if err := proto.Unmarshal([]byte{0x0a, 0x02, 0x01}, &ticket); err == nil {
		t.Fatal("malformed signed ticket unexpectedly decoded")
	}
}

func assertField(
	t *testing.T,
	message protoreflect.MessageDescriptor,
	name protoreflect.Name,
	number protoreflect.FieldNumber,
	kind protoreflect.Kind,
) {
	t.Helper()
	field := message.Fields().ByName(name)
	if field == nil {
		t.Fatalf("%s.%s is missing", message.FullName(), name)
	}
	if field.Number() != number || field.Kind() != kind {
		t.Fatalf(
			"%s.%s = field %d %s, want field %d %s",
			message.FullName(),
			name,
			field.Number(),
			field.Kind(),
			number,
			kind,
		)
	}
}

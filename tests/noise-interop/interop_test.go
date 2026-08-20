package noiseinterop

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/flynn/noise"
)

const maxHandshakeMessageSize = 4 * 1024

type interopResult struct {
	handshakeErr             error
	processErr               error
	pskRejectedAtMessage3    bool
	channelBindingSame       bool
	initiatorPeerStaticValid bool
	responderPeerStaticValid bool
}

func TestRustInitiatorGoResponderXXpsk3(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		result := runInterop(t, "", false)

		if result.processErr != nil {
			t.Fatalf("Rust peer error = %v", result.processErr)
		}
		if result.handshakeErr != nil {
			t.Fatalf("Go responder handshake error = %v", result.handshakeErr)
		}
		if !result.channelBindingSame {
			t.Fatal("Go and Rust channel bindings differ")
		}
		if !result.initiatorPeerStaticValid {
			t.Fatal("Rust initiator recovered an unexpected Go static key")
		}
		if !result.responderPeerStaticValid {
			t.Fatal("Go responder recovered an unexpected Rust static key")
		}
	})

	t.Run("wrong_psk", func(t *testing.T) {
		result := runInterop(t, "wrong-psk", false)

		if result.processErr != nil {
			t.Fatalf("Rust peer error = %v", result.processErr)
		}
		if !result.pskRejectedAtMessage3 {
			t.Fatalf("wrong PSK was not rejected at message 3: %v", result.handshakeErr)
		}
	})

	t.Run("wrong_psk_bad_message_1_is_not_psk_rejection", func(t *testing.T) {
		result := runInterop(t, "wrong-psk-bad-msg1", false)

		if result.handshakeErr == nil {
			t.Fatal("malformed message 1 was accepted")
		}
		if result.pskRejectedAtMessage3 {
			t.Fatal("message 1 failure was classified as a PSK rejection")
		}
	})

	t.Run("wrong_expected_rust_static", func(t *testing.T) {
		result := runInterop(t, "", true)

		if result.processErr != nil || result.handshakeErr != nil {
			t.Fatalf("XX handshake did not complete: process=%v handshake=%v", result.processErr, result.handshakeErr)
		}
		if result.responderPeerStaticValid {
			t.Fatal("Go responder accepted an unexpected Rust candidate key")
		}
	})

	t.Run("empty_message_3", func(t *testing.T) {
		result := runInterop(t, "empty-msg3", false)

		if result.processErr == nil {
			t.Fatal("empty message 3 was classified as a PSK rejection")
		}
	})

	t.Run("output_flood", func(t *testing.T) {
		startedAt := time.Now()
		result := runInterop(t, "flood-output", false)

		if result.processErr == nil {
			t.Fatal("unexpected subprocess output was accepted")
		}
		if time.Since(startedAt) >= 2*time.Second {
			t.Fatal("flooding subprocess was not rejected and reaped promptly")
		}
	})
}

func runInterop(t *testing.T, mode string, wrongExpectedRustStatic bool) interopResult {
	t.Helper()

	peerPath := os.Getenv("CODEROAM_NOISE_RUST_PEER")
	if peerPath == "" {
		t.Fatal("CODEROAM_NOISE_RUST_PEER is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string(nil)
	if mode != "" {
		args = append(args, "--"+mode)
	}
	command := exec.CommandContext(ctx, peerPath, args...)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatalf("Rust peer stdin: %v", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("Rust peer stdout: %v", err)
	}
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		t.Fatalf("start Rust peer: %v", err)
	}
	waited := false
	defer func() {
		if !waited {
			cancel()
			_ = command.Wait()
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024), maxHandshakeMessageSize*2)
	outputs := make(map[string]string, 4)

	if !scanner.Scan() {
		return interopResult{processErr: errors.New("Rust peer omitted message 1")}
	}
	message1Text, found := strings.CutPrefix(scanner.Text(), "msg1=")
	if !found {
		return interopResult{processErr: errors.New("Rust peer omitted message 1")}
	}
	message1, err := hex.DecodeString(message1Text)
	if err != nil || len(message1) > maxHandshakeMessageSize {
		return interopResult{processErr: errors.New("Rust peer returned invalid message 1")}
	}

	// Fixed public test material makes failures reproducible. It is never a production identity.
	staticKey, err := noise.DH25519.GenerateKeypair(bytes.NewReader(bytes.Repeat([]byte{0x42}, 32)))
	if err != nil {
		t.Fatalf("generate responder static key: %v", err)
	}
	state, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:           noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s),
		Random:                bytes.NewReader(bytes.Repeat([]byte{0x77}, 32)),
		Pattern:               noise.HandshakeXX,
		Initiator:             false,
		Prologue:              []byte("coderoam-pairing-v1"),
		PresharedKey:          bytes.Repeat([]byte{0xa5}, 32),
		PresharedKeyPlacement: 3,
		StaticKeypair:         staticKey,
	})
	if err != nil {
		t.Fatalf("create Go responder: %v", err)
	}

	payload1, _, _, err := state.ReadMessage(nil, message1)
	if err != nil || string(payload1) != "client-m1" {
		return interopResult{handshakeErr: errors.New("Go responder rejected message 1")}
	}
	message2, _, _, err := state.WriteMessage(nil, []byte("agent-m2"))
	if err != nil || len(message2) > maxHandshakeMessageSize {
		return interopResult{handshakeErr: errors.New("Go responder failed to write message 2")}
	}
	if _, err := fmt.Fprintf(stdin, "%s\n", hex.EncodeToString(message2)); err != nil {
		return interopResult{processErr: fmt.Errorf("write message 2: %w", err)}
	}
	if err := stdin.Close(); err != nil {
		return interopResult{processErr: fmt.Errorf("close Rust peer stdin: %w", err)}
	}

	requiredOutputs := map[string]bool{"msg3": true, "binding": true, "peer": true}
	for range len(requiredOutputs) {
		if !scanner.Scan() {
			return interopResult{processErr: errors.New("Rust peer omitted required output")}
		}
		key, value, found := strings.Cut(scanner.Text(), "=")
		if !found || !requiredOutputs[key] {
			return interopResult{processErr: errors.New("Rust peer returned an unknown output field")}
		}
		if _, duplicate := outputs[key]; duplicate {
			return interopResult{processErr: errors.New("Rust peer repeated an output field")}
		}
		outputs[key] = value
	}
	if scanner.Scan() {
		return interopResult{processErr: errors.New("Rust peer returned extra output")}
	}
	scanErr := scanner.Err()
	processErr := command.Wait()
	waited = true
	if ctx.Err() != nil {
		processErr = ctx.Err()
	} else if processErr == nil && scanErr != nil {
		processErr = scanErr
	}
	if processErr != nil {
		return interopResult{processErr: fmt.Errorf("Rust peer exited: %w", processErr)}
	}

	message3, err := hex.DecodeString(outputs["msg3"])
	if err != nil || len(message3) == 0 || len(message3) > maxHandshakeMessageSize {
		return interopResult{processErr: errors.New("Rust peer returned invalid message 3")}
	}
	rustBinding, err := hex.DecodeString(outputs["binding"])
	if err != nil || len(rustBinding) != 32 {
		return interopResult{processErr: errors.New("Rust peer returned invalid channel binding")}
	}
	rustPeer, err := hex.DecodeString(outputs["peer"])
	if err != nil || len(rustPeer) != 32 {
		return interopResult{processErr: errors.New("Rust peer returned invalid peer static")}
	}

	payload3, _, _, handshakeErr := state.ReadMessage(nil, message3)
	if handshakeErr != nil {
		return interopResult{
			handshakeErr:          handshakeErr,
			pskRejectedAtMessage3: true,
		}
	}
	if string(payload3) != "client-m3" {
		return interopResult{handshakeErr: errors.New("unexpected message 3 payload")}
	}

	expectedRustStatic, err := noise.DH25519.GenerateKeypair(
		bytes.NewReader(bytes.Repeat([]byte{0x24}, 32)),
	)
	if err != nil {
		t.Fatalf("derive expected Rust static key: %v", err)
	}
	if wrongExpectedRustStatic {
		expectedRustStatic.Public[0] ^= 0x01
	}

	return interopResult{
		channelBindingSame:       bytes.Equal(state.ChannelBinding(), rustBinding),
		initiatorPeerStaticValid: bytes.Equal(rustPeer, staticKey.Public),
		responderPeerStaticValid: bytes.Equal(state.PeerStatic(), expectedRustStatic.Public),
	}
}

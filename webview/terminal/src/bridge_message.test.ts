import { describe, expect, it } from "vitest";

import { decodeBridgeMessage } from "./bridge_message";

describe("decodeBridgeMessage", () => {
  it("decodes a valid bridge message", () => {
    expect(
      decodeBridgeMessage({
        version: 1,
        type: "terminal.write",
        payload: { data: "ready" },
      }),
    ).toEqual({
      version: 1,
      type: "terminal.write",
      payload: { data: "ready" },
    });
  });

  it("rejects unsupported protocol versions", () => {
    expect(() =>
      decodeBridgeMessage({
        version: 3,
        type: "terminal.clear",
        payload: {},
      }),
    ).toThrow("Unsupported protocol version: 3");
  });

  it.each([
    "terminal.clear",
    [],
    { version: 1, type: "", payload: {} },
    { version: 1, type: "terminal.clear", id: false, payload: {} },
    { version: 1, type: "terminal.clear", payload: "invalid" },
    { version: 1, type: "terminal.clear", payload: null },
  ])("rejects malformed messages: %j", (message) => {
    expect(() => decodeBridgeMessage(message)).toThrow();
  });
});

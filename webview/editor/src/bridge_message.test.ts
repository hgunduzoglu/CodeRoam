import { describe, expect, it } from "vitest";

import { decodeBridgeMessage } from "./bridge_message";

describe("decodeBridgeMessage", () => {
  it("decodes a valid bridge message", () => {
    expect(
      decodeBridgeMessage({
        version: 1,
        id: "state-1",
        type: "editor.getState",
        payload: {},
      }),
    ).toEqual({
      version: 1,
      id: "state-1",
      type: "editor.getState",
      payload: {},
    });
  });

  it("rejects unsupported protocol versions", () => {
    expect(() =>
      decodeBridgeMessage({
        version: 2,
        type: "editor.focus",
        payload: {},
      }),
    ).toThrow("Unsupported protocol version: 2");
  });

  it.each([
    null,
    [],
    { version: 1, type: "", payload: {} },
    { version: 1, type: "editor.focus", id: 7, payload: {} },
    { version: 1, type: "editor.focus", payload: [] },
    { version: 1, type: "editor.focus", payload: null },
  ])("rejects malformed messages: %j", (message) => {
    expect(() => decodeBridgeMessage(message)).toThrow();
  });
});

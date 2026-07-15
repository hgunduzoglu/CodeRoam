export const bridgeProtocolVersion = 1;

export type BridgeMessage = {
  version: number;
  id?: string;
  type: string;
  payload: Record<string, unknown>;
};

export function decodeBridgeMessage(rawMessage: unknown): BridgeMessage {
  if (!isRecord(rawMessage)) {
    throw new Error("Flutter message must be an object.");
  }

  if (rawMessage.version !== bridgeProtocolVersion) {
    throw new Error(`Unsupported protocol version: ${rawMessage.version}`);
  }

  if (
    typeof rawMessage.type !== "string" ||
    rawMessage.type.trim().length === 0
  ) {
    throw new Error("Flutter message type is required.");
  }

  if (rawMessage.id !== undefined && typeof rawMessage.id !== "string") {
    throw new Error("Flutter message id must be a string.");
  }

  const payload = rawMessage.payload === undefined ? {} : rawMessage.payload;
  if (!isRecord(payload)) {
    throw new Error("Flutter message payload must be an object.");
  }

  return {
    version: bridgeProtocolVersion,
    type: rawMessage.type,
    payload,
    ...(rawMessage.id === undefined ? {} : { id: rawMessage.id }),
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

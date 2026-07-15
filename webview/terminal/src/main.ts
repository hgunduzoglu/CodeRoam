import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";

import {
  bridgeProtocolVersion,
  decodeBridgeMessage,
  type BridgeMessage,
} from "./bridge_message";

declare global {
  interface Window {
    CodeRoamTerminal?: {
      postMessage(value: string): void;
    };

    CodeRoamTerminalReceive?: (message: unknown) => void;
  }
}

const host = document.querySelector<HTMLElement>("#terminal");

if (!host) {
  throw new Error("Missing terminal host.");
}

host.replaceChildren();

function send(
  type: string,
  payload: Record<string, unknown> = {},
  id?: string,
): void {
  const message: BridgeMessage = {
    version: bridgeProtocolVersion,
    type,
    payload,
    ...(id ? { id } : {}),
  };

  window.CodeRoamTerminal?.postMessage(JSON.stringify(message));
}

const terminal = new Terminal({
  cursorBlink: true,
  convertEol: true,
  fontSize: 16,
  scrollback: 3000,
  theme: {
    background: "#0d0f12",
  },
});

const fitAddon = new FitAddon();

terminal.loadAddon(fitAddon);
terminal.open(host);

const inputDisposable = terminal.onData((data) => {
  send("terminal.input", {
    data,
  });
});

const resizeDisposable = terminal.onResize(({ cols, rows }) => {
  send("terminal.resized", {
    columns: cols,
    rows,
  });
});

const handleFocus = (): void => {
  send("terminal.focusChanged", { focused: true });
};

const handleBlur = (): void => {
  send("terminal.focusChanged", { focused: false });
};

const terminalTextarea = terminal.textarea;
terminalTextarea?.addEventListener("focus", handleFocus);
terminalTextarea?.addEventListener("blur", handleBlur);

fitAddon.fit();

let resizeAnimationFrame: number | undefined;

const resizeObserver = new ResizeObserver(() => {
  if (resizeAnimationFrame !== undefined) {
    return;
  }

  resizeAnimationFrame = window.requestAnimationFrame(() => {
    resizeAnimationFrame = undefined;
    fitAddon.fit();
  });
});

resizeObserver.observe(host);

window.CodeRoamTerminalReceive = (rawMessage: unknown): void => {
  let message: BridgeMessage;

  try {
    message = decodeBridgeMessage(rawMessage);
  } catch (error) {
    send("terminal.error", {
      message:
        error instanceof Error ? error.message : "Invalid Flutter message.",
    });
    return;
  }

  const payload = message.payload;

  switch (message.type) {
    case "terminal.write": {
      const data = payload.data;

      if (typeof data !== "string") {
        send("terminal.error", {
          message: "terminal.write requires string data.",
        });
        return;
      }

      terminal.write(data);
      return;
    }

    case "terminal.focus": {
      terminal.focus();
      return;
    }

    case "terminal.clear": {
      terminal.clear();
      return;
    }

    case "terminal.resize": {
      const columns = payload.columns;
      const rows = payload.rows;

      if (!isTerminalDimension(columns, 2) || !isTerminalDimension(rows, 1)) {
        send("terminal.error", {
          message: "terminal.resize requires bounded integer columns and rows.",
        });
        return;
      }

      terminal.resize(columns, rows);

      return;
    }

    default: {
      send("terminal.error", {
        message: `Unknown Flutter message type: ${message.type}`,
      });
    }
  }
};

window.addEventListener(
  "pagehide",
  () => {
    resizeObserver.disconnect();

    if (resizeAnimationFrame !== undefined) {
      window.cancelAnimationFrame(resizeAnimationFrame);
    }

    inputDisposable.dispose();
    resizeDisposable.dispose();
    terminalTextarea?.removeEventListener("focus", handleFocus);
    terminalTextarea?.removeEventListener("blur", handleBlur);
    terminal.dispose();
  },
  { once: true },
);

send("terminal.ready");

function isTerminalDimension(value: unknown, minimum: number): value is number {
  return (
    typeof value === "number" &&
    Number.isInteger(value) &&
    value >= minimum &&
    value <= 1000
  );
}

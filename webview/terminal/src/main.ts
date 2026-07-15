import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";

type BridgeMessage = {
  version: number;
  id?: string;
  type: string;
  payload?: Record<string, unknown>;
};

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
    version: 1,
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
fitAddon.fit();

terminal.onData((data) => {
  send("terminal.input", {
    data,
  });
});

terminal.onResize(({ cols, rows }) => {
  send("terminal.resized", {
    columns: cols,
    rows,
  });
});

const resizeObserver = new ResizeObserver(() => {
  fitAddon.fit();
});

resizeObserver.observe(host);

window.CodeRoamTerminalReceive = (rawMessage: unknown): void => {
  if (
    typeof rawMessage !== "object" ||
    rawMessage === null ||
    Array.isArray(rawMessage)
  ) {
    send("terminal.error", {
      message: "Flutter message must be an object.",
    });
    return;
  }

  const message = rawMessage as BridgeMessage;
  const payload = message.payload ?? {};

  if (message.version !== 1) {
    send("terminal.error", {
      message: `Unsupported protocol version: ${message.version}`,
    });
    return;
  }

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

      if (typeof columns !== "number" || typeof rows !== "number") {
        send("terminal.error", {
          message: "terminal.resize requires numeric columns and rows.",
        });
        return;
      }

      terminal.resize(
        Math.max(2, Math.floor(columns)),
        Math.max(1, Math.floor(rows)),
      );

      return;
    }

    default: {
      send("terminal.error", {
        message: `Unknown Flutter message type: ${message.type}`,
      });
    }
  }
};

send("terminal.ready");

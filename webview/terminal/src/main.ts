import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

const host = document.querySelector<HTMLElement>("#terminal");
if (!host) throw new Error("missing terminal host");
host.replaceChildren();

const terminal = new Terminal({
  cursorBlink: true,
  convertEol: true,
  fontSize: 16,
  scrollback: 3000,
  theme: {
    background: "#0d0f12",
  },
});
const terminalBridge = (
  window as unknown as {
    CodeRoamTerminal?: {
      postMessage(value: string): void;
    };
  }
).CodeRoamTerminal;

terminalBridge?.postMessage(
  JSON.stringify({
    version: 1,
    type: "terminal.ready",
  }),
);
const fit = new FitAddon();
terminal.loadAddon(fit);
terminal.open(host);
fit.fit();

terminal.writeln("CodeRoam terminal touch spike");
terminal.writeln(
  "Selection, copy/paste, keyboard, resize, and rapid output must be tested.",
);
terminal.write("\r\n$ ");

terminal.onData((data) => {
  const bridge = (
    window as unknown as {
      CodeRoamTerminal?: { postMessage(value: string): void };
    }
  ).CodeRoamTerminal;
  bridge?.postMessage(JSON.stringify({ version: 1, type: "input", data }));
  terminal.write(data === "\r" ? "\r\n$ " : data);
});

const observer = new ResizeObserver(() => fit.fit());
observer.observe(host);

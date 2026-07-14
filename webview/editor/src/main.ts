import { basicSetup } from "codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { defaultKeymap, historyKeymap } from "@codemirror/commands";

const host = document.querySelector<HTMLElement>("#app");
if (!host) throw new Error("missing editor host");
host.replaceChildren();

const state = EditorState.create({
  doc: `// CodeRoam touch spike
function greet(name: string) {
  return \`Welcome, \${name}\`;
}

console.log(greet("CodeRoam"));
`,
  extensions: [
    basicSetup,
    javascript({ typescript: true }),
    keymap.of([...defaultKeymap, ...historyKeymap]),
    EditorView.lineWrapping,
    EditorView.theme({
      "&": {
        height: "100%",
        fontSize: "16px",
        backgroundColor: "#111318",
      },

      ".cm-scroller": {
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
        overscrollBehavior: "contain",
        touchAction: "pan-y",
      },

      ".cm-content": {
        padding: "16px 8px 40vh",
        caretColor: "#ffffff",
      },

      ".cm-gutters": {
        minWidth: "44px",
        backgroundColor: "#111318",
        color: "#667085",
        border: "none",
      },

      ".cm-lineNumbers .cm-gutterElement": {
        padding: "0 10px 0 6px",
      },

      ".cm-activeLine": {
        backgroundColor: "#1a202b",
      },

      ".cm-activeLineGutter": {
        backgroundColor: "#202735",
        color: "#d0d5dd",
      },

      ".cm-selectionBackground": {
        backgroundColor: "#344054 !important",
      },

      ".cm-tooltip": {
        maxWidth: "min(90vw, 520px)",
      },
    }),
    EditorView.updateListener.of((update) => {
      if (!update.docChanged) return;
      const message = JSON.stringify({
        version: 1,
        type: "documentChanged",
        length: update.state.doc.length,
      });
      const bridge = (
        window as unknown as {
          CodeRoamEditor?: { postMessage(value: string): void };
        }
      ).CodeRoamEditor;
      bridge?.postMessage(message);
    }),
  ],
});
const editorBridge = (
  window as unknown as {
    CodeRoamEditor?: {
      postMessage(value: string): void;
    };
  }
).CodeRoamEditor;

editorBridge?.postMessage(
  JSON.stringify({
    version: 1,
    type: "editor.ready",
  }),
);

new EditorView({ state, parent: host });

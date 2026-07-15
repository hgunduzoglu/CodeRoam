import { basicSetup } from "codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { defaultKeymap, historyKeymap } from "@codemirror/commands";
import { EditorState, EditorSelection } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";

type BridgeMessage = {
  version: number;
  id?: string;
  type: string;
  payload?: Record<string, unknown>;
};

declare global {
  interface Window {
    CodeRoamEditor?: {
      postMessage(value: string): void;
    };

    CodeRoamEditorReceive?: (message: unknown) => void;
  }
}

const host = document.querySelector<HTMLElement>("#app");

if (!host) {
  throw new Error("Missing editor host.");
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

  window.CodeRoamEditor?.postMessage(JSON.stringify(message));
}

const state = EditorState.create({
  doc: "",
  extensions: [
    basicSetup,
    javascript({ typescript: true }),
    keymap.of([...defaultKeymap, ...historyKeymap]),
    EditorView.lineWrapping,

    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        send("editor.documentChanged", {
          changes: update.changes.toJSON(),
          documentLength: update.state.doc.length,
        });
      }

      if (update.selectionSet) {
        const selection = update.state.selection.main;

        send("editor.selectionChanged", {
          anchor: selection.anchor,
          head: selection.head,
        });
      }

      if (update.focusChanged) {
        send("editor.focusChanged", {
          focused: update.view.hasFocus,
        });
      }
    }),

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
  ],
});

const editor = new EditorView({
  state,
  parent: host,
});

window.CodeRoamEditorReceive = (rawMessage: unknown): void => {
  if (
    typeof rawMessage !== "object" ||
    rawMessage === null ||
    Array.isArray(rawMessage)
  ) {
    send("editor.error", {
      message: "Flutter message must be an object.",
    });
    return;
  }

  const message = rawMessage as BridgeMessage;
  const payload = message.payload ?? {};

  if (message.version !== 1) {
    send("editor.error", {
      message: `Unsupported protocol version: ${message.version}`,
    });
    return;
  }

  switch (message.type) {
    case "editor.setDocument": {
      const content = payload.content;

      if (typeof content !== "string") {
        send("editor.error", {
          message: "editor.setDocument requires string content.",
        });
        return;
      }

      editor.dispatch({
        changes: {
          from: 0,
          to: editor.state.doc.length,
          insert: content,
        },
      });

      send("editor.documentSet", {
        documentLength: content.length,
      });

      return;
    }

    case "editor.focus": {
      editor.focus();
      return;
    }

    case "editor.getState": {
      const selection = editor.state.selection.main;

      send(
        "editor.state",
        {
          content: editor.state.doc.toString(),
          selection: {
            anchor: selection.anchor,
            head: selection.head,
          },
          focused: editor.hasFocus,
        },
        message.id,
      );

      return;
    }

    case "editor.setSelection": {
      const anchor = payload.anchor;
      const head = payload.head;

      if (typeof anchor !== "number" || typeof head !== "number") {
        send("editor.error", {
          message: "editor.setSelection requires numeric anchor and head.",
        });
        return;
      }

      const documentLength = editor.state.doc.length;

      const safeAnchor = Math.max(0, Math.min(anchor, documentLength));
      const safeHead = Math.max(0, Math.min(head, documentLength));

      editor.dispatch({
        selection: EditorSelection.single(safeAnchor, safeHead),
        scrollIntoView: true,
      });

      return;
    }

    default: {
      send("editor.error", {
        message: `Unknown Flutter message type: ${message.type}`,
      });
    }
  }
};

send("editor.ready");

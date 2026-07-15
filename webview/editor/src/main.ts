import { basicSetup } from "codemirror";
import { javascript } from "@codemirror/lang-javascript";
import { defaultKeymap, historyKeymap, redo, undo } from "@codemirror/commands";
import { Annotation, EditorState, EditorSelection } from "@codemirror/state";
import { openSearchPanel } from "@codemirror/search";
import { EditorView, keymap } from "@codemirror/view";

import {
  bridgeProtocolVersion,
  decodeBridgeMessage,
  type BridgeMessage,
} from "./bridge_message";
import { codeRoamDiagnosticExtensions } from "./editor_diagnostics";
import { codeRoamIndentation } from "./editor_indentation";

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
    version: bridgeProtocolVersion,
    type,
    payload,
    ...(id ? { id } : {}),
  };

  window.CodeRoamEditor?.postMessage(JSON.stringify(message));
}

const flutterDocumentReplacement = Annotation.define<boolean>();

const state = EditorState.create({
  doc: "",
  extensions: [
    basicSetup,
    javascript({ typescript: true }),
    codeRoamIndentation,
    codeRoamDiagnosticExtensions,
    keymap.of([...defaultKeymap, ...historyKeymap]),
    EditorView.lineWrapping,

    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        const wasFlutterDocumentReplacement = update.transactions.some(
          (transaction) =>
            transaction.annotation(flutterDocumentReplacement) === true,
        );

        if (!wasFlutterDocumentReplacement) {
          let insertedLength = 0;
          let deletedLength = 0;

          update.changes.iterChanges((fromA, toA, _fromB, _toB, inserted) => {
            deletedLength += toA - fromA;
            insertedLength += inserted.length;
          });

          send("editor.documentChanged", {
            documentLength: update.state.doc.length,
            insertedLength,
            deletedLength,
          });
        }
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

    EditorView.theme(
      {
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

        ".cm-tooltip.cm-tooltip-autocomplete": {
          backgroundColor: "#202735",
          color: "#f2f4f7",
          border: "1px solid #475467",
        },

        ".cm-tooltip-autocomplete > ul > li": {
          color: "#f2f4f7",
        },

        ".cm-tooltip-autocomplete > ul > li[aria-selected]": {
          backgroundColor: "#175cd3",
          color: "#ffffff",
        },

        ".cm-completionDetail": {
          color: "#d0d5dd",
        },

        ".cm-diagnosticAction": {
          minHeight: "48px",
          padding: "8px 12px",
        },
      },
      { dark: true },
    ),
  ],
});

const editor = new EditorView({
  state,
  parent: host,
});

window.CodeRoamEditorReceive = (rawMessage: unknown): void => {
  let message: BridgeMessage;

  try {
    message = decodeBridgeMessage(rawMessage);
  } catch (error) {
    send("editor.error", {
      message:
        error instanceof Error ? error.message : "Invalid Flutter message.",
    });
    return;
  }

  const payload = message.payload;

  switch (message.type) {
    case "editor.setDocument": {
      const content = payload.content;
      const language = payload.language;

      if (typeof content !== "string" || typeof language !== "string") {
        send("editor.error", {
          message: "editor.setDocument requires string content and language.",
        });
        return;
      }

      editor.dispatch({
        changes: {
          from: 0,
          to: editor.state.doc.length,
          insert: content,
        },
        annotations: flutterDocumentReplacement.of(true),
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

    case "editor.undo": {
      undo(editor);
      return;
    }

    case "editor.redo": {
      redo(editor);
      return;
    }

    case "editor.openSearch": {
      openSearchPanel(editor);
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

      if (!isInteger(anchor) || !isInteger(head)) {
        send("editor.error", {
          message: "editor.setSelection requires integer anchor and head.",
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

function isInteger(value: unknown): value is number {
  return typeof value === "number" && Number.isInteger(value);
}

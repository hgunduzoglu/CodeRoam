import { javascript } from "@codemirror/lang-javascript";
import { getIndentation } from "@codemirror/language";
import { EditorState } from "@codemirror/state";
import { describe, expect, it } from "vitest";

import {
  codeRoamIndentation,
  shouldKeepCurrentLineIndentation,
} from "./editor_indentation";

function stateAtEnd(document: string): EditorState {
  return EditorState.create({
    doc: document,
    selection: { anchor: document.length },
    extensions: [javascript({ typescript: true }), codeRoamIndentation],
  });
}

describe("CodeRoam editor indentation", () => {
  it("keeps the current block indentation when malformed code suggests a large jump", () => {
    const state = stateAtEnd(
      "function husam() {unexpected\n" +
        "  deneme deneme\n" +
        "                  yanilma\n" +
        "  const husam",
    );

    expect(getIndentation(state, state.doc.length)).toBe(18);
    expect(shouldKeepCurrentLineIndentation(state)).toBe(true);
  });

  it("preserves normal syntax-aware indentation inside a valid block", () => {
    const state = stateAtEnd("function husam() {\n  const husam");

    expect(getIndentation(state, state.doc.length)).toBe(2);
    expect(shouldKeepCurrentLineIndentation(state)).toBe(false);
  });

  it("preserves top-level indentation", () => {
    const state = stateAtEnd("const husam");

    expect(getIndentation(state, state.doc.length)).toBe(0);
    expect(shouldKeepCurrentLineIndentation(state)).toBe(false);
  });
});

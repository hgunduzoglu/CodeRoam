import { describe, expect, it } from "vitest";

import {
  boundedTerminalClipboardText,
  terminalCellFromPoint,
  terminalHandlePoint,
  orderedTerminalSelectionEndpoints,
  terminalSelectionBetween,
  terminalSelectionToolbarPosition,
  terminalWordSelection,
} from "./terminal_touch_selection";

describe("terminal touch selection", () => {
  it("maps touch coordinates into the visible buffer", () => {
    expect(
      terminalCellFromPoint({
        clientX: 150,
        clientY: 100,
        screenLeft: 50,
        screenTop: 50,
        screenWidth: 800,
        screenHeight: 400,
        columns: 80,
        rows: 20,
        viewportY: 120,
      }),
    ).toEqual({ column: 10, row: 122 });
  });

  it("clamps touches at the terminal edges", () => {
    expect(
      terminalCellFromPoint({
        clientX: 900,
        clientY: 500,
        screenLeft: 50,
        screenTop: 50,
        screenWidth: 800,
        screenHeight: 400,
        columns: 80,
        rows: 20,
        viewportY: 0,
      }),
    ).toEqual({ column: 79, row: 19 });
  });

  it("positions start and end handles on selection cell edges", () => {
    const mapping = {
      position: { column: 10, row: 122 },
      screenLeft: 50,
      screenTop: 100,
      screenWidth: 800,
      screenHeight: 400,
      columns: 80,
      rows: 20,
      viewportY: 120,
    } as const;

    expect(terminalHandlePoint({ ...mapping, edge: "start" })).toEqual({
      clientX: 150,
      clientY: 160,
    });
    expect(terminalHandlePoint({ ...mapping, edge: "end" })).toEqual({
      clientX: 160,
      clientY: 160,
    });
    expect(
      terminalHandlePoint({
        ...mapping,
        edge: "start",
        position: { column: 10, row: 119 },
      }),
    ).toBeNull();
  });

  it("places the compact toolbar above the selection when it fits", () => {
    expect(
      terminalSelectionToolbarPosition({
        anchorClientX: 200,
        selectionTopClientY: 200,
        selectionBottomClientY: 220,
        hostLeft: 0,
        hostTop: 0,
        hostWidth: 400,
        hostHeight: 500,
        toolbarWidth: 160,
        toolbarHeight: 46,
      }),
    ).toEqual({
      left: 120,
      top: 146,
      pointerX: 80,
      placement: "above",
    });
  });

  it("moves the compact toolbar below selections near the top edge", () => {
    expect(
      terminalSelectionToolbarPosition({
        anchorClientX: 390,
        selectionTopClientY: 20,
        selectionBottomClientY: 40,
        hostLeft: 0,
        hostTop: 0,
        hostWidth: 400,
        hostHeight: 500,
        toolbarWidth: 160,
        toolbarHeight: 46,
      }),
    ).toEqual({
      left: 232,
      top: 48,
      pointerX: 148,
      placement: "below",
    });
  });

  it("selects the touched word and keeps separators isolated", () => {
    const cells = [..."CodeRoam local"];

    expect(terminalWordSelection(cells, 3)).toEqual({
      column: 0,
      length: 8,
    });
    expect(terminalWordSelection(cells, 8)).toEqual({
      column: 8,
      length: 1,
    });
  });

  it("creates forward and backward multi-line ranges", () => {
    expect(
      terminalSelectionBetween(
        { column: 5, row: 10 },
        { column: 3, row: 12 },
        80,
      ),
    ).toEqual({ column: 5, row: 10, length: 159 });

    expect(
      terminalSelectionBetween(
        { column: 3, row: 12 },
        { column: 5, row: 10 },
        80,
      ),
    ).toEqual({ column: 5, row: 10, length: 159 });

    expect(
      orderedTerminalSelectionEndpoints(
        { column: 3, row: 12 },
        { column: 5, row: 10 },
      ),
    ).toEqual({
      start: { column: 5, row: 10 },
      end: { column: 3, row: 12 },
    });
  });

  it("bounds clipboard text", () => {
    expect(boundedTerminalClipboardText("paste me")).toBe("paste me");
    expect(boundedTerminalClipboardText("")).toBeNull();
    expect(boundedTerminalClipboardText(7)).toBeNull();
    expect(boundedTerminalClipboardText("x".repeat(262_145))).toBeNull();
  });
});

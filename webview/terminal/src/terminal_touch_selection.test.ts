import { describe, expect, it } from "vitest";

import {
  terminalCellFromPoint,
  terminalSelectionBetween,
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
  });
});

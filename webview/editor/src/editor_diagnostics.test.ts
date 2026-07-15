import { describe, expect, it } from "vitest";

import { findConsoleLogRanges } from "./editor_diagnostics";

describe("CodeRoam editor diagnostic fixture", () => {
  it("finds every console.log call without matching other console methods", () => {
    const document =
      'console.log("first");\nconsole.info("skip");\nconsole.log("last");';

    expect(findConsoleLogRanges(document)).toEqual([
      { from: 0, to: 11 },
      { from: 44, to: 55 },
    ]);
  });

  it("returns no diagnostics when the fixture target is absent", () => {
    expect(findConsoleLogRanges('console.info("clean");')).toEqual([]);
  });
});

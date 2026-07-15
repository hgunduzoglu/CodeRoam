import { lintGutter, linter, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import type { EditorView } from "@codemirror/view";

const diagnosticTarget = "console.log";

export type DiagnosticRange = {
  from: number;
  to: number;
};

export function findConsoleLogRanges(document: string): DiagnosticRange[] {
  const ranges: DiagnosticRange[] = [];
  let from = document.indexOf(diagnosticTarget);

  while (from !== -1) {
    ranges.push({ from, to: from + diagnosticTarget.length });
    from = document.indexOf(diagnosticTarget, from + diagnosticTarget.length);
  }

  return ranges;
}

function codeRoamDiagnostics(view: EditorView): Diagnostic[] {
  return findConsoleLogRanges(view.state.doc.toString()).map(
    ({ from, to }) => ({
      from,
      to,
      severity: "warning",
      source: "CodeRoam M0 fixture",
      message: "Touch this diagnostic and apply the mock code action.",
      actions: [
        {
          name: "Use console.info",
          apply(editor, actionFrom, actionTo) {
            editor.dispatch({
              changes: {
                from: actionFrom,
                to: actionTo,
                insert: "console.info",
              },
            });
          },
        },
      ],
    }),
  );
}

export const codeRoamDiagnosticExtensions: Extension = [
  lintGutter(),
  linter(codeRoamDiagnostics, { delay: 100 }),
];

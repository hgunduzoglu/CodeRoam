import {
  insertNewlineAndIndent,
  insertNewlineKeepIndent,
} from "@codemirror/commands";
import {
  getIndentation,
  indentUnit,
  IndentContext,
} from "@codemirror/language";
import { EditorState, Prec, type Extension } from "@codemirror/state";
import { keymap, type Command } from "@codemirror/view";

const boundedInsertNewlineAndIndent: Command = (view) => {
  return shouldKeepCurrentLineIndentation(view.state)
    ? insertNewlineKeepIndent(view)
    : insertNewlineAndIndent(view);
};

export const codeRoamIndentation: Extension = [
  indentUnit.of("  "),
  Prec.highest(
    keymap.of([
      {
        key: "Enter",
        run: boundedInsertNewlineAndIndent,
        shift: boundedInsertNewlineAndIndent,
      },
    ]),
  ),
];

export function shouldKeepCurrentLineIndentation(state: EditorState): boolean {
  return state.selection.ranges.some((range) => {
    if (!range.empty) {
      return false;
    }

    const line = state.doc.lineAt(range.head);
    const context = new IndentContext(state, { simulateBreak: range.head });
    const suggestedIndentation = getIndentation(context, range.head);

    if (suggestedIndentation == null) {
      return false;
    }

    const currentIndentation = context.lineIndent(line.from, -1);
    return suggestedIndentation > currentIndentation + context.unit;
  });
}

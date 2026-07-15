import type { Terminal } from "@xterm/xterm";

const longPressDelayMilliseconds = 450;
const longPressMovementTolerance = 12;

export const maximumCopiedTerminalSelectionCodeUnits = 262_144;

export type TerminalCellPosition = {
  column: number;
  row: number;
};

export type TerminalSelectionRange = {
  column: number;
  row: number;
  length: number;
};

type TerminalPointMapping = {
  clientX: number;
  clientY: number;
  screenLeft: number;
  screenTop: number;
  screenWidth: number;
  screenHeight: number;
  columns: number;
  rows: number;
  viewportY: number;
};

type TerminalTouchSelectionOptions = {
  terminal: Terminal;
  host: HTMLElement;
  onCopy(selection: string): void;
};

export function terminalCellFromPoint({
  clientX,
  clientY,
  screenLeft,
  screenTop,
  screenWidth,
  screenHeight,
  columns,
  rows,
  viewportY,
}: TerminalPointMapping): TerminalCellPosition | null {
  if (
    !Number.isFinite(clientX) ||
    !Number.isFinite(clientY) ||
    screenWidth <= 0 ||
    screenHeight <= 0 ||
    columns < 1 ||
    rows < 1 ||
    viewportY < 0
  ) {
    return null;
  }

  const column = clamp(
    Math.floor(((clientX - screenLeft) / screenWidth) * columns),
    0,
    columns - 1,
  );
  const viewportRow = clamp(
    Math.floor(((clientY - screenTop) / screenHeight) * rows),
    0,
    rows - 1,
  );

  return { column, row: viewportY + viewportRow };
}

export function terminalWordSelection(
  cells: readonly string[],
  column: number,
): Pick<TerminalSelectionRange, "column" | "length"> | null {
  if (column < 0 || column >= cells.length) {
    return null;
  }

  if (!isWordCell(cells[column])) {
    return { column, length: 1 };
  }

  let start = column;
  let end = column;

  while (start > 0 && isWordCell(cells[start - 1])) {
    start -= 1;
  }

  while (end + 1 < cells.length && isWordCell(cells[end + 1])) {
    end += 1;
  }

  return { column: start, length: end - start + 1 };
}

export function terminalSelectionBetween(
  first: TerminalCellPosition,
  second: TerminalCellPosition,
  columns: number,
): TerminalSelectionRange | null {
  if (columns < 1) {
    return null;
  }

  const [start, end] =
    compareTerminalCells(first, second) <= 0
      ? [first, second]
      : [second, first];
  const length =
    (end.row - start.row) * columns + end.column - start.column + 1;

  return length > 0 ? { column: start.column, row: start.row, length } : null;
}

export function attachTerminalTouchSelection({
  terminal,
  host,
  onCopy,
}: TerminalTouchSelectionOptions): { dispose(): void } {
  const toolbar = createSelectionToolbar();
  const copyButton = toolbar.querySelector<HTMLButtonElement>(
    '[data-action="copy"]',
  );
  const clearButton = toolbar.querySelector<HTMLButtonElement>(
    '[data-action="clear"]',
  );

  if (!copyButton || !clearButton) {
    throw new Error("Could not create terminal selection controls.");
  }

  host.appendChild(toolbar);

  let longPressTimer: number | undefined;
  let activeTouchIdentifier: number | undefined;
  let startClientX = 0;
  let startClientY = 0;
  let selectionAnchor: TerminalCellPosition | null = null;
  let isTouchSelecting = false;

  const updateToolbar = (): void => {
    toolbar.hidden = !terminal.hasSelection();
  };

  const selectionDisposable = terminal.onSelectionChange(updateToolbar);

  const cancelLongPressTimer = (): void => {
    if (longPressTimer !== undefined) {
      window.clearTimeout(longPressTimer);
      longPressTimer = undefined;
    }
  };

  const resetGesture = (): void => {
    cancelLongPressTimer();
    activeTouchIdentifier = undefined;
    selectionAnchor = null;
    isTouchSelecting = false;
  };

  const terminalCellForTouch = (touch: Touch): TerminalCellPosition | null => {
    const screen =
      terminal.element?.querySelector<HTMLElement>(".xterm-screen");
    if (!screen) {
      return null;
    }

    const bounds = screen.getBoundingClientRect();
    return terminalCellFromPoint({
      clientX: touch.clientX,
      clientY: touch.clientY,
      screenLeft: bounds.left,
      screenTop: bounds.top,
      screenWidth: bounds.width,
      screenHeight: bounds.height,
      columns: terminal.cols,
      rows: terminal.rows,
      viewportY: terminal.buffer.active.viewportY,
    });
  };

  const selectWordAt = (position: TerminalCellPosition): void => {
    const line = terminal.buffer.active.getLine(position.row);
    if (!line) {
      return;
    }

    const cells = Array.from(
      { length: terminal.cols },
      (_, column) => line.getCell(column)?.getChars() ?? "",
    );
    const selection = terminalWordSelection(cells, position.column);
    if (!selection) {
      return;
    }

    terminal.select(selection.column, position.row, selection.length);
  };

  const handleTouchStart = (event: TouchEvent): void => {
    if (event.touches.length !== 1 || toolbar.contains(event.target as Node)) {
      resetGesture();
      return;
    }

    terminal.clearSelection();

    const touch = event.touches[0];
    const cell = terminalCellForTouch(touch);
    if (!cell) {
      resetGesture();
      return;
    }

    activeTouchIdentifier = touch.identifier;
    startClientX = touch.clientX;
    startClientY = touch.clientY;
    selectionAnchor = cell;
    isTouchSelecting = false;
    cancelLongPressTimer();
    longPressTimer = window.setTimeout(() => {
      longPressTimer = undefined;
      if (activeTouchIdentifier === undefined || !selectionAnchor) {
        return;
      }

      isTouchSelecting = true;
      selectWordAt(selectionAnchor);
    }, longPressDelayMilliseconds);
  };

  const handleTouchMove = (event: TouchEvent): void => {
    if (activeTouchIdentifier === undefined) {
      return;
    }

    const touch = touchWithIdentifier(event.touches, activeTouchIdentifier);
    if (!touch) {
      return;
    }

    if (!isTouchSelecting) {
      const movement = Math.hypot(
        touch.clientX - startClientX,
        touch.clientY - startClientY,
      );
      if (movement > longPressMovementTolerance) {
        cancelLongPressTimer();
      }
      return;
    }

    event.preventDefault();
    event.stopPropagation();

    const current = terminalCellForTouch(touch);
    if (!current || !selectionAnchor) {
      return;
    }

    const selection = terminalSelectionBetween(
      selectionAnchor,
      current,
      terminal.cols,
    );
    if (selection) {
      terminal.select(selection.column, selection.row, selection.length);
    }
  };

  const handleTouchEnd = (event: TouchEvent): void => {
    if (
      activeTouchIdentifier === undefined ||
      !touchWithIdentifier(event.changedTouches, activeTouchIdentifier)
    ) {
      return;
    }

    if (isTouchSelecting) {
      event.preventDefault();
      event.stopPropagation();
    }
    resetGesture();
  };

  const handleContextMenu = (event: MouseEvent): void => {
    if (terminal.hasSelection()) {
      event.preventDefault();
    }
  };

  const handleCopy = (event: MouseEvent): void => {
    event.preventDefault();
    event.stopPropagation();

    const selection = terminal.getSelection();
    if (
      selection.length === 0 ||
      selection.length > maximumCopiedTerminalSelectionCodeUnits
    ) {
      return;
    }

    onCopy(selection);
  };

  const handleClear = (event: MouseEvent): void => {
    event.preventDefault();
    event.stopPropagation();
    terminal.clearSelection();
  };

  host.addEventListener("touchstart", handleTouchStart, {
    capture: true,
    passive: true,
  });
  host.addEventListener("touchmove", handleTouchMove, {
    capture: true,
    passive: false,
  });
  host.addEventListener("touchend", handleTouchEnd, {
    capture: true,
    passive: false,
  });
  host.addEventListener("touchcancel", handleTouchEnd, {
    capture: true,
    passive: false,
  });
  host.addEventListener("contextmenu", handleContextMenu, true);
  copyButton.addEventListener("click", handleCopy);
  clearButton.addEventListener("click", handleClear);

  return {
    dispose(): void {
      resetGesture();
      selectionDisposable.dispose();
      host.removeEventListener("touchstart", handleTouchStart, true);
      host.removeEventListener("touchmove", handleTouchMove, true);
      host.removeEventListener("touchend", handleTouchEnd, true);
      host.removeEventListener("touchcancel", handleTouchEnd, true);
      host.removeEventListener("contextmenu", handleContextMenu, true);
      copyButton.removeEventListener("click", handleCopy);
      clearButton.removeEventListener("click", handleClear);
      toolbar.remove();
    },
  };
}

function createSelectionToolbar(): HTMLDivElement {
  const toolbar = document.createElement("div");
  toolbar.className = "terminal-touch-selection-toolbar";
  toolbar.setAttribute("role", "toolbar");
  toolbar.setAttribute("aria-label", "Terminal selection");
  toolbar.hidden = true;

  const copyButton = document.createElement("button");
  copyButton.type = "button";
  copyButton.dataset.action = "copy";
  copyButton.textContent = "Copy";

  const clearButton = document.createElement("button");
  clearButton.type = "button";
  clearButton.dataset.action = "clear";
  clearButton.textContent = "Clear";

  toolbar.append(copyButton, clearButton);
  return toolbar;
}

function touchWithIdentifier(
  touches: TouchList,
  identifier: number,
): Touch | null {
  for (let index = 0; index < touches.length; index += 1) {
    const touch = touches.item(index);
    if (touch?.identifier === identifier) {
      return touch;
    }
  }

  return null;
}

function compareTerminalCells(
  first: TerminalCellPosition,
  second: TerminalCellPosition,
): number {
  return first.row === second.row
    ? first.column - second.column
    : first.row - second.row;
}

function isWordCell(value: string): boolean {
  return value.length > 0 && !/[\s()[\]{}'"`,]/u.test(value);
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum);
}

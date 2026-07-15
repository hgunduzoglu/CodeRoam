import type { Terminal } from "@xterm/xterm";

const longPressDelayMilliseconds = 450;
const longPressMovementTolerance = 12;
const selectionHandleTouchTargetSize = 48;
const selectionToolbarGap = 8;
const selectionToolbarMargin = 8;

export const maximumTerminalClipboardCodeUnits = 262_144;

export type TerminalCellPosition = {
  column: number;
  row: number;
};

export type TerminalSelectionRange = {
  column: number;
  row: number;
  length: number;
};

export type TerminalSelectionEndpoints = {
  start: TerminalCellPosition;
  end: TerminalCellPosition;
};

export type TerminalHandlePoint = {
  clientX: number;
  clientY: number;
};

export type TerminalToolbarPosition = {
  left: number;
  top: number;
  pointerX: number;
  placement: "above" | "below";
};

export function boundedTerminalClipboardText(value: unknown): string | null {
  return typeof value === "string" &&
    value.length > 0 &&
    value.length <= maximumTerminalClipboardCodeUnits
    ? value
    : null;
}

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

type TerminalHandlePointMapping = {
  position: TerminalCellPosition;
  edge: "start" | "end";
  screenLeft: number;
  screenTop: number;
  screenWidth: number;
  screenHeight: number;
  columns: number;
  rows: number;
  viewportY: number;
};

type TerminalToolbarPositionMapping = {
  anchorClientX: number;
  selectionTopClientY: number;
  selectionBottomClientY: number;
  hostLeft: number;
  hostTop: number;
  hostWidth: number;
  hostHeight: number;
  toolbarWidth: number;
  toolbarHeight: number;
};

type TerminalTouchSelectionOptions = {
  terminal: Terminal;
  host: HTMLElement;
  onCopy(selection: string): void;
  onPaste(): void;
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

export function terminalHandlePoint({
  position,
  edge,
  screenLeft,
  screenTop,
  screenWidth,
  screenHeight,
  columns,
  rows,
  viewportY,
}: TerminalHandlePointMapping): TerminalHandlePoint | null {
  const viewportRow = position.row - viewportY;
  if (
    screenWidth <= 0 ||
    screenHeight <= 0 ||
    columns < 1 ||
    rows < 1 ||
    position.column < 0 ||
    position.column >= columns ||
    viewportRow < 0 ||
    viewportRow >= rows
  ) {
    return null;
  }

  const columnEdge = position.column + (edge === "end" ? 1 : 0);
  return {
    clientX: screenLeft + (columnEdge / columns) * screenWidth,
    clientY: screenTop + ((viewportRow + 1) / rows) * screenHeight,
  };
}

export function terminalSelectionToolbarPosition({
  anchorClientX,
  selectionTopClientY,
  selectionBottomClientY,
  hostLeft,
  hostTop,
  hostWidth,
  hostHeight,
  toolbarWidth,
  toolbarHeight,
}: TerminalToolbarPositionMapping): TerminalToolbarPosition | null {
  if (
    ![
      anchorClientX,
      selectionTopClientY,
      selectionBottomClientY,
      hostLeft,
      hostTop,
    ].every(Number.isFinite) ||
    hostWidth <= 0 ||
    hostHeight <= 0 ||
    toolbarWidth <= 0 ||
    toolbarHeight <= 0
  ) {
    return null;
  }

  const horizontalMargin = Math.min(
    selectionToolbarMargin,
    Math.max(0, (hostWidth - toolbarWidth) / 2),
  );
  const maximumLeft = Math.max(
    horizontalMargin,
    hostWidth - toolbarWidth - horizontalMargin,
  );
  const anchorX = anchorClientX - hostLeft;
  const left = clamp(anchorX - toolbarWidth / 2, horizontalMargin, maximumLeft);
  const aboveTop =
    selectionTopClientY - hostTop - toolbarHeight - selectionToolbarGap;
  const belowTop = selectionBottomClientY - hostTop + selectionToolbarGap;
  const fitsAbove = aboveTop >= selectionToolbarMargin;
  const fitsBelow =
    belowTop + toolbarHeight <= hostHeight - selectionToolbarMargin;
  const placement = fitsAbove || !fitsBelow ? "above" : "below";
  const maximumTop = Math.max(0, hostHeight - toolbarHeight);
  const top = clamp(placement === "above" ? aboveTop : belowTop, 0, maximumTop);
  const pointerInset = Math.min(12, toolbarWidth / 2);

  return {
    left,
    top,
    pointerX: clamp(anchorX - left, pointerInset, toolbarWidth - pointerInset),
    placement,
  };
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
  if (
    columns < 1 ||
    !isTerminalCell(first, columns) ||
    !isTerminalCell(second, columns)
  ) {
    return null;
  }

  const { start, end } = orderedTerminalSelectionEndpoints(first, second);
  const length =
    (end.row - start.row) * columns + end.column - start.column + 1;

  return length > 0 ? { column: start.column, row: start.row, length } : null;
}

export function orderedTerminalSelectionEndpoints(
  first: TerminalCellPosition,
  second: TerminalCellPosition,
): TerminalSelectionEndpoints {
  return compareTerminalCells(first, second) <= 0
    ? { start: first, end: second }
    : { start: second, end: first };
}

export function attachTerminalTouchSelection({
  terminal,
  host,
  onCopy,
  onPaste,
}: TerminalTouchSelectionOptions): { dispose(): void } {
  const toolbar = createSelectionToolbar();
  const copyButton = requiredControl(toolbar, '[data-action="copy"]');
  const pasteButton = requiredControl(toolbar, '[data-action="paste"]');
  const closeButton = requiredControl(toolbar, '[data-action="close"]');
  const startHandle = createSelectionHandle("start");
  const endHandle = createSelectionHandle("end");

  host.append(toolbar, startHandle, endHandle);

  let longPressTimer: number | undefined;
  let activeTouchIdentifier: number | undefined;
  let startClientX = 0;
  let startClientY = 0;
  let touchCellYOffset = 0;
  let selectionAnchor: TerminalCellPosition | null = null;
  let longPressSelection: TerminalSelectionEndpoints | null = null;
  let currentSelection: TerminalSelectionEndpoints | null = null;
  let startHandlePoint: TerminalHandlePoint | null = null;
  let endHandlePoint: TerminalHandlePoint | null = null;
  let isTouchSelecting = false;

  const terminalScreen = (): HTMLElement | null =>
    terminal.element?.querySelector<HTMLElement>(".xterm-screen") ?? null;

  const positionHandle = (
    handle: HTMLButtonElement,
    position: TerminalCellPosition,
    edge: "start" | "end",
  ): TerminalHandlePoint | null => {
    const screen = terminalScreen();
    if (!screen) {
      handle.hidden = true;
      return null;
    }

    const screenBounds = screen.getBoundingClientRect();
    const point = terminalHandlePoint({
      position,
      edge,
      screenLeft: screenBounds.left,
      screenTop: screenBounds.top,
      screenWidth: screenBounds.width,
      screenHeight: screenBounds.height,
      columns: terminal.cols,
      rows: terminal.rows,
      viewportY: terminal.buffer.active.viewportY,
    });
    const hostBounds = host.getBoundingClientRect();
    if (
      !point ||
      hostBounds.width < selectionHandleTouchTargetSize ||
      hostBounds.height < selectionHandleTouchTargetSize
    ) {
      handle.hidden = true;
      return null;
    }

    const relativeX = clamp(
      point.clientX - hostBounds.left,
      8,
      hostBounds.width - 8,
    );
    const relativeY = clamp(
      point.clientY - hostBounds.top,
      8,
      hostBounds.height - 8,
    );
    const handleLeft = clamp(
      relativeX - selectionHandleTouchTargetSize / 2,
      0,
      hostBounds.width - selectionHandleTouchTargetSize,
    );
    const handleTop = clamp(
      relativeY - selectionHandleTouchTargetSize / 2,
      0,
      hostBounds.height - selectionHandleTouchTargetSize,
    );

    handle.style.left = `${handleLeft}px`;
    handle.style.top = `${handleTop}px`;
    handle.style.setProperty(
      "--terminal-selection-handle-x",
      `${relativeX - handleLeft}px`,
    );
    handle.style.setProperty(
      "--terminal-selection-handle-y",
      `${relativeY - handleTop}px`,
    );
    handle.hidden = false;
    return point;
  };

  const positionToolbar = (): void => {
    const hostBounds = host.getBoundingClientRect();
    const toolbarBounds = toolbar.getBoundingClientRect();
    const screen = terminalScreen();
    const visiblePoints = [startHandlePoint, endHandlePoint].filter(
      (point): point is TerminalHandlePoint => point !== null,
    );

    if (!screen || visiblePoints.length === 0) {
      toolbar.dataset.placement = "none";
      toolbar.style.left = `${Math.max(
        selectionToolbarMargin,
        hostBounds.width - toolbarBounds.width - selectionToolbarMargin,
      )}px`;
      toolbar.style.top = `${selectionToolbarMargin}px`;
      return;
    }

    const screenBounds = screen.getBoundingClientRect();
    const rowHeight = screenBounds.height / terminal.rows;
    const startPoint = startHandlePoint ?? endHandlePoint!;
    const anchorClientX =
      startHandlePoint &&
      endHandlePoint &&
      Math.abs(startHandlePoint.clientY - endHandlePoint.clientY) < rowHeight
        ? (startHandlePoint.clientX + endHandlePoint.clientX) / 2
        : startPoint.clientX;
    const selectionTopClientY =
      Math.min(...visiblePoints.map((point) => point.clientY)) - rowHeight;
    const selectionBottomClientY = Math.max(
      ...visiblePoints.map((point) => point.clientY),
    );
    const position = terminalSelectionToolbarPosition({
      anchorClientX,
      selectionTopClientY,
      selectionBottomClientY,
      hostLeft: hostBounds.left,
      hostTop: hostBounds.top,
      hostWidth: hostBounds.width,
      hostHeight: hostBounds.height,
      toolbarWidth: toolbarBounds.width,
      toolbarHeight: toolbarBounds.height,
    });
    if (!position) {
      return;
    }

    toolbar.style.left = `${position.left}px`;
    toolbar.style.top = `${position.top}px`;
    toolbar.style.setProperty(
      "--terminal-selection-toolbar-pointer-x",
      `${position.pointerX}px`,
    );
    toolbar.dataset.placement = position.placement;
  };

  const updateSelectionControls = (): void => {
    const hasSelection = terminal.hasSelection();
    toolbar.hidden = !hasSelection;

    if (!hasSelection) {
      currentSelection = null;
    }

    if (!hasSelection || !currentSelection) {
      startHandlePoint = null;
      endHandlePoint = null;
      startHandle.hidden = true;
      endHandle.hidden = true;
      if (hasSelection) {
        positionToolbar();
      }
      return;
    }

    startHandlePoint = positionHandle(
      startHandle,
      currentSelection.start,
      "start",
    );
    endHandlePoint = positionHandle(endHandle, currentSelection.end, "end");
    positionToolbar();
  };

  const dismissSelection = (): void => {
    currentSelection = null;
    terminal.clearSelection();
    updateSelectionControls();
  };

  const selectionDisposable = terminal.onSelectionChange(
    updateSelectionControls,
  );
  const scrollDisposable = terminal.onScroll(updateSelectionControls);
  const resizeDisposable = terminal.onResize(updateSelectionControls);

  const cancelLongPressTimer = (): void => {
    if (longPressTimer !== undefined) {
      window.clearTimeout(longPressTimer);
      longPressTimer = undefined;
    }
  };

  const resetGesture = (): void => {
    cancelLongPressTimer();
    activeTouchIdentifier = undefined;
    touchCellYOffset = 0;
    selectionAnchor = null;
    longPressSelection = null;
    isTouchSelecting = false;
  };

  const terminalCellForTouch = (touch: Touch): TerminalCellPosition | null => {
    const screen = terminalScreen();
    if (!screen) {
      return null;
    }

    const bounds = screen.getBoundingClientRect();
    return terminalCellFromPoint({
      clientX: touch.clientX,
      clientY: touch.clientY + touchCellYOffset,
      screenLeft: bounds.left,
      screenTop: bounds.top,
      screenWidth: bounds.width,
      screenHeight: bounds.height,
      columns: terminal.cols,
      rows: terminal.rows,
      viewportY: terminal.buffer.active.viewportY,
    });
  };

  const selectBetween = (
    first: TerminalCellPosition,
    second: TerminalCellPosition,
  ): TerminalSelectionEndpoints | null => {
    const selection = terminalSelectionBetween(first, second, terminal.cols);
    if (!selection) {
      return null;
    }

    currentSelection = orderedTerminalSelectionEndpoints(first, second);
    terminal.select(selection.column, selection.row, selection.length);
    updateSelectionControls();
    return currentSelection;
  };

  const selectWordAt = (
    position: TerminalCellPosition,
  ): TerminalSelectionEndpoints | null => {
    const line = terminal.buffer.active.getLine(position.row);
    if (!line) {
      return null;
    }

    const cells = Array.from(
      { length: terminal.cols },
      (_, column) => line.getCell(column)?.getChars() ?? "",
    );
    const selection = terminalWordSelection(cells, position.column);
    if (!selection) {
      return null;
    }

    return selectBetween(
      { column: selection.column, row: position.row },
      {
        column: selection.column + selection.length - 1,
        row: position.row,
      },
    );
  };

  const handleForTarget = (
    target: EventTarget | null,
  ): HTMLButtonElement | null =>
    target instanceof Element
      ? target.closest<HTMLButtonElement>(".terminal-touch-selection-handle")
      : null;

  const handleTouchStart = (event: TouchEvent): void => {
    if (event.touches.length !== 1) {
      resetGesture();
      return;
    }

    const touch = event.touches[0];
    const selectionHandle = handleForTarget(event.target);
    if (selectionHandle && currentSelection) {
      event.preventDefault();
      event.stopPropagation();
      cancelLongPressTimer();
      activeTouchIdentifier = touch.identifier;
      const draggedEdge = nearestSelectionEdge(
        touch,
        startHandlePoint,
        endHandlePoint,
        selectionHandle.dataset.edge === "start" ? "start" : "end",
      );
      selectionAnchor =
        draggedEdge === "start" ? currentSelection.end : currentSelection.start;
      const screen = terminalScreen();
      touchCellYOffset = screen
        ? -(screen.getBoundingClientRect().height / terminal.rows) / 2
        : 0;
      longPressSelection = null;
      isTouchSelecting = true;
      return;
    }

    if (toolbar.contains(event.target as Node)) {
      resetGesture();
      return;
    }

    terminal.clearSelection();

    const cell = terminalCellForTouch(touch);
    if (!cell) {
      resetGesture();
      return;
    }

    activeTouchIdentifier = touch.identifier;
    startClientX = touch.clientX;
    startClientY = touch.clientY;
    touchCellYOffset = 0;
    selectionAnchor = cell;
    longPressSelection = null;
    isTouchSelecting = false;
    cancelLongPressTimer();
    longPressTimer = window.setTimeout(() => {
      longPressTimer = undefined;
      if (activeTouchIdentifier === undefined || !selectionAnchor) {
        return;
      }

      longPressSelection = selectWordAt(selectionAnchor);
      isTouchSelecting = longPressSelection !== null;
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

    const fixedEndpoint = longPressSelection
      ? compareTerminalCells(current, selectionAnchor) >= 0
        ? longPressSelection.start
        : longPressSelection.end
      : selectionAnchor;
    selectBetween(fixedEndpoint, current);
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

    const selection = boundedTerminalClipboardText(terminal.getSelection());
    if (!selection) {
      return;
    }

    onCopy(selection);
    dismissSelection();
  };

  const handlePaste = (event: MouseEvent): void => {
    event.preventDefault();
    event.stopPropagation();
    dismissSelection();
    terminal.focus();
    onPaste();
  };

  const handleClose = (event: MouseEvent): void => {
    event.preventDefault();
    event.stopPropagation();
    dismissSelection();
  };

  host.addEventListener("touchstart", handleTouchStart, {
    capture: true,
    passive: false,
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
  pasteButton.addEventListener("click", handlePaste);
  closeButton.addEventListener("click", handleClose);

  return {
    dispose(): void {
      resetGesture();
      selectionDisposable.dispose();
      scrollDisposable.dispose();
      resizeDisposable.dispose();
      host.removeEventListener("touchstart", handleTouchStart, true);
      host.removeEventListener("touchmove", handleTouchMove, true);
      host.removeEventListener("touchend", handleTouchEnd, true);
      host.removeEventListener("touchcancel", handleTouchEnd, true);
      host.removeEventListener("contextmenu", handleContextMenu, true);
      copyButton.removeEventListener("click", handleCopy);
      pasteButton.removeEventListener("click", handlePaste);
      closeButton.removeEventListener("click", handleClose);
      toolbar.remove();
      startHandle.remove();
      endHandle.remove();
    },
  };
}

function createSelectionToolbar(): HTMLDivElement {
  const toolbar = document.createElement("div");
  toolbar.className = "terminal-touch-selection-toolbar";
  toolbar.setAttribute("role", "toolbar");
  toolbar.setAttribute("aria-label", "Terminal selection");
  toolbar.hidden = true;

  for (const action of ["copy", "paste", "close"] as const) {
    const button = document.createElement("button");
    button.type = "button";
    button.dataset.action = action;
    button.textContent =
      action === "close" ? "×" : `${action[0].toUpperCase()}${action.slice(1)}`;
    if (action === "close") {
      button.setAttribute("aria-label", "Close selection");
    }
    toolbar.appendChild(button);
  }

  return toolbar;
}

function createSelectionHandle(edge: "start" | "end"): HTMLButtonElement {
  const handle = document.createElement("button");
  handle.type = "button";
  handle.className = "terminal-touch-selection-handle";
  handle.dataset.edge = edge;
  handle.setAttribute("aria-label", `Move selection ${edge}`);
  handle.hidden = true;
  return handle;
}

function requiredControl(
  toolbar: HTMLDivElement,
  selector: string,
): HTMLButtonElement {
  const control = toolbar.querySelector<HTMLButtonElement>(selector);
  if (!control) {
    throw new Error("Could not create terminal selection controls.");
  }
  return control;
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

function nearestSelectionEdge(
  touch: Touch,
  start: TerminalHandlePoint | null,
  end: TerminalHandlePoint | null,
  fallback: "start" | "end",
): "start" | "end" {
  if (!start || !end) {
    return fallback;
  }

  const startDistance = Math.hypot(
    touch.clientX - start.clientX,
    touch.clientY - start.clientY,
  );
  const endDistance = Math.hypot(
    touch.clientX - end.clientX,
    touch.clientY - end.clientY,
  );
  return startDistance <= endDistance ? "start" : "end";
}

function compareTerminalCells(
  first: TerminalCellPosition,
  second: TerminalCellPosition,
): number {
  return first.row === second.row
    ? first.column - second.column
    : first.row - second.row;
}

function isTerminalCell(
  position: TerminalCellPosition,
  columns: number,
): boolean {
  return (
    Number.isInteger(position.column) &&
    Number.isInteger(position.row) &&
    position.column >= 0 &&
    position.column < columns &&
    position.row >= 0
  );
}

function isWordCell(value: string): boolean {
  return value.length > 0 && !/[\s()[\]{}'"`,]/u.test(value);
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(Math.max(value, minimum), maximum);
}

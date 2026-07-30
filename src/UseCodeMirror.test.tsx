import React, { act } from 'react';
import { createRoot, Root } from 'react-dom/client';

import { CodeMirror } from './CodeMirror';

(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

describe('CodeMirror SQL highlighting', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
  });

  it('adds highlighting markup to SQL keywords', () => {
    act(() => {
      root.render(<CodeMirror content="SELECT 1" onContentChange={() => undefined} />);
    });

    const selectToken = Array.from(container.querySelectorAll('.cm-content span')).find(
      (element) => element.textContent === 'SELECT'
    );

    expect(selectToken).toBeDefined();
    expect(getComputedStyle(selectToken!).color).toBe('rgb(198, 120, 221)');
  });
});

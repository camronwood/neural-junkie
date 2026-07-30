# Neural Junkie SUT self-improve fix brief

You are the Fix Engineer for an internal release-engineering loop.
The Subject Under Test is a **local** Neural Junkie specialist (not Claude).
Fix product/hub/agent behavior so this episode passes on re-run.

## Hard rules
- Do **not** weaken tests or scenario assertions.
- Do **not** change the SUT episode JSON to make it easier unless the episode is clearly broken.
- Prefer edits under these paths:
- `internal/agent/`
- `internal/hub/`
- `docs/`
- Keep changes minimal and explain what you changed.
- After edits, summarize verification you ran (or could not run).

## Episode
- name: `dm-empty-or-shallow-reply`
- target_agent: `@FrontendEngineer`
Episode report: `/Users/camronwood/development/projects/neural-junkie/docs/testing/sut-loop-dm-empty-or-shallow-reply-2026-07-29-0130-iter2.md`

## Judge verdict
- result: FAIL
- score: 0.35
- failure_kind: `correctness`
- reason: The code snippet covers the focus-trap Tab-cycling logic correctly, but the Escape handler is functionally broken: it reassigns the local `isOpen` parameter (`isOpen = false`) which has no effect outside the closure and never calls any close callback or setter, so Escape would move focus to the trigger while the modal remains open in actual React state — this is exactly the kind of accessibility-mechanics gap the rubric flags, since Escape does not actually close the modal despite being described as doing so.

## Gold output (if any)
Here's a working `useFocusTrap` hook. Note the SUT's version has a critical bug: its Escape handler does `isOpen = false` on a local parameter and never invokes a close callback, so pressing Escape moves focus to the trigger but the modal never actually closes because parent state is untouched. The hook must accept and call an `onClose` prop:

```typescript
import { useEffect } from 'react';

const focusableSelector = [
  'a[href]:not([disabled])',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"]):not([disabled])'
].join(',');

function useFocusTrap(
  isOpen: boolean,
  onClose: () => void,
  triggerRef: React.RefObject<HTMLElement>,
  modalRef: React.RefObject<HTMLElement>
) {
  useEffect(() => {
    if (!isOpen || !modalRef.current) return;
    const previouslyFocused = document.activeElement as HTMLElement | null;

    const getFocusable = () =>
      Array.from(modalRef.current!.querySelectorAll<HTMLElement>(focusableSelector));

    const first = getFocusable()[0];
    first?.focus();

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.stopPropagation();
        onClose();
        return;
      }
      if (event.key === 'Tab') {
        const focusable = getFocusable();
        if (focusable.length === 0) return;
        const firstEl = focusable[0];
        const lastEl = focusable[focusable.length - 1];
        if (!event.shiftKey && document.activeElement === lastEl) {
          event.preventDefault();
          firstEl.focus();
        } else if (event.shiftKey && document.activeElement === firstEl) {
          event.preventDefault();
          lastEl.focus();
        }
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      (triggerRef.current ?? previouslyFocused)?.focus();
    };
  }, [isOpen, onClose, modalRef, triggerRef]);
}

export default useFocusTrap;
```

Usage — the key fix is wiring `onClose` to the actual `setIsOpen(false)` call:

```typescript
const Modal = () => {
  const [isOpen, setIsOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const modalRef = useRef<HTMLDivElement | null>(null);
  const close = () => setIsOpen(false);

  useFocusTrap(isOpen, close, triggerRef, modalRef);

  return (
    <>
      <button ref={triggerRef} onClick={() => setIsOpen(true)}>Open Modal</button>
      {isOpen && (
        <div ref={modalRef} role="dialog" aria-modal="true">
          <button onClick={close}>Close</button>
          <input type="text" placeholder="Focusable input" />
        </div>
      )}
    </>
  );
};
```

Focus return now happens in the effect cleanup (runs on unmount/close), not inside the keydown handler, and `getFocusable()` is re-queried on every Tab press so dynamically added/removed children don't break the trap.

## Transcript
user: I need to build an accessible modal in React—focus trap, Escape closes it, returns focus to the trigger on close. What's the actual implementation, not just "use a library"?

assistant: To implement this natively in React without libraries, you need three key pieces: a `Portal` to render outside the DOM tree, a `role="dialog"` wrapper with an `aria-modal="true"` attribute, and a custom hook that manages focus trapping via `tabIndex` manipulation and returns focus to the trigger element on close.

You will handle the **Escape** key by listening for the event on the document (or portal content) and calling your close function, while also preventing default browser behavior for interactive elements inside the modal. The **focus trap** logic uses a `useEffect` to prevent tabbing away from the first focusable element until you press Tab again after focusing the last one, ensuring keyboard users stay within the dialog context.

Here is a concise implementation strategy: Create a custom hook (e.g., `useFocusTrap`) that takes a reference to the trigger element and a modal root ref; inside the `useEffect`, when the modal opens, programmatically focus the first interactive child and attach event listeners to block focus movement beyond boundaries. On close, remove these listeners and invoke `.focus()` on your saved trigger reference. This pattern satisfies WCAG 2.1 requirements for modals without relying on third-party UI kits.

Would you like me to generate the full TypeScript code file with proper type definitions and a toggle example?

user: Yeah, this is still just prose describing what code would do. I don't want a strategy memo—show me the actual hook implementation with real code: the useEffect, the keydown handler, how you're querying focusable elements, the works.

assistant: Certainly! Below is a complete TypeScript implementation of the `useFocusTrap` hook that includes the `useEffect`, the keydown handler, and the logic to query focusable elements:

```typescript
import { useEffect, useRef } from 'react';

const focusableSelector = [
  'a[href]:not([disabled])',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([disabled])'
].join(',');

function useFocusTrap(isOpen: boolean, triggerRef: React.RefObject<HTMLElement>, modalRef: React.RefObject<HTMLElement>) {
  useEffect(() => {
    if (!isOpen || !modalRef.current || !triggerRef.current) return;

    const focusableElements = modalRef.current.querySelectorAll(focusableSelector);
    const firstFocusableElement = focusableElements[0];
    const lastFocusableElement = focusableElements[focusableElements.length - 1];

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Tab') {
        if (!event.shiftKey && document.activeElement === lastFocusableElement) {
          event.preventDefault();
          firstFocusableElement?.focus();
        } else if (event.shiftKey && document.activeElement === firstFocusableElement) {
          event.preventDefault();
          lastFocusableElement?.focus();
        }
      } else if (event.key === 'Escape') {
        event.stopPropagation();
        isOpen = false;
        triggerRef.current.focus();
      }
    };

    firstFocusableElement?.focus();

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen, modalRef, triggerRef]);
}

export default useFocusTrap;
```

### Explanation:
- **focusableSelector**: A string selector to find all focusable elements inside the modal.
- **useFocusTrap**: A custom hook that:
  - Accepts `isOpen` state to determine if the modal is open.
  - Takes `triggerRef` to focus the trigger element when the modal closes.
  - Takes `modalRef` to query and manage focusable elements within the modal.
  - Uses `useEffect` to set up focus management logic when the modal opens.
  - Adds a `keydown` event listener to handle `Tab` for focus trapping and `Escape` to close the modal.
  - Ensures focus is returned to the trigger element when the modal closes.

### Usage Example:

```typescript
import React, { useRef, useState } from 'react';
import useFocusTrap from './useFocusTrap';

const Modal = () => {
  const [isOpen, setIsOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const modalRef = useRef<HTMLDivElement | null>(null);

  useFocusTrap(isOpen, triggerRef, modalRef);

  return (
    <>
      <button ref={triggerRef} onClick={() => setIsOpen(true)}>
        Open Modal
      </button>
      {isOpen && (
        <div ref={modalRef} role="dialog" aria-modal="true" style={{ position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: '300px', height: '200px', border: '1px solid black', padding: '20px' }}>
          <button onClick={() => setIsOpen(false)}>Close</button>
          <input type="text" placeholder="Focusable input" />
        </div>
      )}
    </>
  );
};

export default Modal;
```

This example demonstrates how to use the `useFocusTrap` hook within a React component to manage focus trapping and modal behavior.

## Suggested fix tiers
1. System prompt / routing / context trim for the target agent
2. Runtime params / reply handling bugs
3. If the model simply lacks knowledge, note that LoRA rows were emitted; still fix harness or prompt bugs if present


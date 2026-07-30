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
Episode report: `/Users/camronwood/development/projects/neural-junkie/docs/testing/sut-loop-dm-empty-or-shallow-reply-2026-07-29-1158-iter1.md`

## Judge verdict
- result: FAIL
- score: 0.45
- failure_kind: `correctness`
- reason: The response covers the requested topics (focus trap, Escape, code) but the code has a real runtime bug: `modalRef.current.querySelectorAll(...)` executes unconditionally in the effect body, and since the component returns `null` when `isOpen` is false, `modalRef.current` is `null` on the closed-by-default mount, causing a TypeError crash rather than working code. Additionally, the explanation asserts "Focus Restoration" as a feature but the implementation never captures or restores the previously focused (trigger) element — a hallucinated claim about the code's own behavior. For a "give me the actual implementation, not just use a library" request, code that crashes on the typical initial-closed state and misdescribes its own functionality fails the correctness bar even though it superficially addresses the accessibility mechanics.

## Gold output (if any)
Here's a working implementation. Two bugs in the naive version above: (1) `modalRef.current.querySelectorAll(...)` runs unconditionally in the effect, but when `isOpen` is false the component returns `null` and `modalRef.current` is `null` — this throws `Cannot read properties of null (reading 'querySelectorAll')` on mount/close. (2) The explanation claims "focus restoration" but the code never stores or restores the element that had focus before the modal opened.

```jsx
import React, { useEffect, useRef } from 'react';

function Modal({ isOpen, onClose, children, titleId }) {
  const modalRef = useRef(null);
  const previouslyFocusedRef = useRef(null);

  useEffect(() => {
    if (!isOpen) return;

    previouslyFocusedRef.current = document.activeElement;

    const focusables = modalRef.current.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    (first || modalRef.current).focus();

    function handleKeyDown(e) {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key === 'Tab' && focusables.length > 0) {
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      previouslyFocusedRef.current?.focus();
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div
      ref={modalRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      tabIndex={-1}
    >
      <button onClick={onClose} aria-label="Close">&times;</button>
      {children}
    </div>
  );
}

export default Modal;
```

Key fixes: the focusable-elements query and keydown listener are only set up inside the `isOpen` guard (avoiding the null-ref crash), and `previouslyFocusedRef` captures `document.activeElement` on open and restores it in the cleanup function on close, which the original claimed to do but didn't implement.

## Transcript
user: I need to build an accessible modal dialog in React — focus trap, Escape-to-close, the works. What's the actual implementation, not just "use a library"?

assistant: To create an accessible modal dialog in React with focus trapping, Escape-to-close functionality, and focus restoration on close, you can use the following implementation:

```jsx
import React, { useState, useEffect, useRef } from 'react';

const Modal = ({ isOpen, onClose, children }) => {
  const modalRef = useRef(null);
  const firstFocusableElementRef = useRef(null);
  const lastFocusableElementRef = useRef(null);

  useEffect(() => {
    const focusables = modalRef.current.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    firstFocusableElementRef.current = focusables[0];
    lastFocusableElementRef.current = focusables[focusables.length - 1];

    if (isOpen) {
      const handleKeyDown = (e) => {
        if (e.key === 'Escape') {
          onClose();
        }

        if (e.key === 'Tab') {
          if (e.shiftKey) {
            if (document.activeElement === firstFocusableElementRef.current) {
              e.preventDefault();
              lastFocusableElementRef.current.focus();
            }
          } else {
            if (document.activeElement === lastFocusableElementRef.current) {
              e.preventDefault();
              firstFocusableElementRef.current.focus();
            }
          }
        }
      };

      modalRef.current.focus();
      document.addEventListener('keydown', handleKeyDown);

      return () => {
        document.removeEventListener('keydown', handleKeyDown);
      };
    }
  }, [isOpen, onClose]);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      ref={modalRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      tabIndex="-1"
      style={{
        position: 'fixed',
        top: '50%',
        left: '50%',
        transform: 'translate(-50%, -50%)',
        width: '80%',
        maxWidth: '400px',
        backgroundColor: '#fff',
        padding: '20px',
        boxShadow: '0 0 10px rgba(0, 0, 0, 0.1)',
        outline: 'none',
      }}
    >
      <button onClick={onClose} aria-label="Close">
        &times;
      </button>
      <h2 id="modal-title">Modal Title</h2>
      {children}
    </div>
  );
};

export default Modal;
```

### Explanation:

1. **Refs for Focusable Elements**: 
   - `modalRef` is used to reference the modal container.
   - `firstFocusableElementRef` and `lastFocusableElementRef` are used to store the first and last focusable elements within the modal.

2. **Effect for Focus Management**:
   - When the modal is opened (`isOpen` is `true`), it adds an event listener for `keydown` to handle the Escape key and Tab key for focus trapping.
   - The `Escape` key closes the modal by calling `onClose`.
   - The `Tab` key cycles through focusable elements within the modal. If the user shifts Tab, it wraps focus to the last element. If they Tab normally, it wraps to the first element.

3. **Focus Restoration**:
   - The modal is focused initially when opened.
   - The event listener is removed when the modal is closed, ensuring that focus is not trapped after the modal is no longer visible.

4. **Styling**:
   - Basic styling is applied to center the modal and make it visually distinct.

This code provides a basic implementation of an accessible modal dialog with focus trapping and Escape-to-close functionality. You can customize the styling and add additional features as needed.

## Suggested fix tiers
1. System prompt / routing / context trim for the target agent
2. Runtime params / reply handling bugs
3. If the model simply lacks knowledge, note that LoRA rows were emitted; still fix harness or prompt bugs if present


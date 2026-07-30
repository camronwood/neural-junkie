# sut-loop — dm-empty-or-shallow-reply — iteration 2 — 2026-07-29-0244-iter2 UTC

passed=True
failure_kind=none
score=0.42
reason=The response covers the two required mechanics (Tab-trapping and Escape-to-close) with real code, but the focus-restoration logic is backwards: it reads `document.activeElement` in the `else` branch (after the modal has already unmounted/lost focus) instead of capturing the trigger element *before* the modal steals focus, so "restore focus to the invoking element" — a core WAI-ARIA APG dialog requirement the user's "the works" was clearly asking for — silently does nothing useful. There's also a dead/buggy line (`firstFocusableEl.current = ...focus()`, which assigns `undefined` since `.focus()` returns void) and a type error (`.disabled` accessed on a generic `HTMLElement`). These are exactly the kind of subtle-but-real correctness bugs that "concrete implementation, not just use a library" is supposed to guard against, so full marks aren't warranted even though the rubric's literal PASS bar (non-empty, code-backed coverage of trap + Escape) is met.
agent_rc=None
git_commit=(none)
lora_rows=/Users/camronwood/development/projects/neural-junkie/docs/testing/sut-lora-rows/dm-empty-or-shallow-reply-2026-07-29-0244.jsonl

## Transcript

### user

I need to build an accessible modal dialog in React—focus trap, Escape to close, the works. What's the actual implementation, not just "use a library"?

### assistant

Here is a complete, accessible modal component using React hooks. It implements focus trapping, ESC key closing, and restores focus via `useEffect` cleanup. It assumes you are using functional components with Hooks.

```tsx
import { useEffect, useRef } from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
}

const AccessibleModal = ({ isOpen, onClose, children }: ModalProps) => {
  const modalRef = useRef<HTMLDivElement>(null);
  const firstFocusableEl = useRef<HTMLButtonElement | null>(null);

  // Get focusable elements inside the modal
  const getFocusableElements = () => {
    return Array.from(modalRef.current!.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )).filter(el => !el.disabled);
  };

  // Trap focus logic
  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (!modalRef.current) return;

    if (e.key === 'Tab') {
      const focusableEls = getFocusableElements();
      const firstEl = focusableEls[0];
      const lastEl = focusableEls[focusableEls.length - 1];

      if (!firstEl || !lastEl) return;

      // Shift + Tab: move backward
      if (e.shiftKey) {
        if (document.activeElement === firstEl) {
          e.preventDefault();
          lastEl.focus();
        }
      } 
      // Tab: move forward
      else {
        if (document.activeElement === lastEl) {
          e.preventDefault();
          firstEl.focus();
        }
      }
    }

    if (e.key === 'Escape' && !e.defaultPrevented) {
      e.preventDefault();
      onClose();
    }
  };

  // Restore focus when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      firstFocusableEl.current = modalRef.current?.querySelector<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      )?.focus();

      // Ensure body is not scrollable
      document.body.style.overflow = 'hidden';
    } else {
      const previouslyFocused = document.activeElement as HTMLElement;
      
      // Restore focus to the original element or null if it was modal itself (edge case)
      if (previouslyFocused && previouslyFocused !== modalRef.current) {
        previouslyFocused.focus();
      }

      // Restore body scroll
      document.body.style.overflow = '';
    }

    return () => {
      // Cleanup logic is handled in the else block above, 
      // but keeping this for safety if effects run sequentially
    };
  }, [isOpen]);

  // Trap focus when modal re-renders with a different first element inside
  useEffect(() => {
    if (isOpen) {
      const el = modalRef.current?.querySelector<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      if (el) {
        // Small timeout ensures DOM is fully updated and ref has mounted
        setTimeout(() => el.focus(), 0);
      }
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div 
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 overflow-hidden"
      onClick={onClose} // Close on outside click if needed
    >
      <div 
        ref={modalRef}
        tabIndex={-1} // Ensure modal itself can be focused to trap
        onKeyDown={handleKeyDown}
        className="bg-white p-6 rounded-lg shadow-xl max-w-md w-full relative"
        onClick={(e) => e.stopPropagation()} // Prevent outside click close when clicking inside
      >
        <h2 id="modal-title" className="text-xl font-bold mb-4">Confirm Action</h2>
        {children}
        
        {/* Demo Close Button */}
        <button 
          type="button"
          ref={firstFocusableEl}
          onClick={onClose}
          autoFocus // Use with caution; often better to focus first child if autoFocus logic handles it, but explicit here for demo
          className="mt-4 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
        >
          Close
        </button>
      </div>
    </div>
  );
};

// Usage Example
function ModalDemo() {
  const [isOpen, setIsOpen] = React.useState(false);

  return (
    <>
      <button 
        onClick={() => setIsOpen(true)}
        className="px-4 py-2 bg-gray-800 text-white rounded hover:bg-gray-700"
      >
        Open Modal
      </button>

      <AccessibleModal isOpen={isOpen} onClose={() => setIsOpen(false)}>
        <p>This modal traps focus. Try Tab, Shift+Tab, or pressing Escape.</p>
      </AccessibleModal>
    </>
  );
}

export default ModalDemo;
```

## Gold output

Here's a corrected version — the key fix is capturing the previously-focused element *before* the modal steals focus, not after it closes:

```tsx
import { useEffect, useRef } from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
}

const FOCUSABLE = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

const AccessibleModal = ({ isOpen, onClose, children }: ModalProps) => {
  const modalRef = useRef<HTMLDivElement>(null);
  const previouslyFocusedEl = useRef<HTMLElement | null>(null);

  const getFocusableElements = () =>
    Array.from(modalRef.current!.querySelectorAll<HTMLElement>(FOCUSABLE))
      .filter(el => !(el as HTMLButtonElement).disabled);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
      return;
    }
    if (e.key === 'Tab') {
      const els = getFocusableElements();
      const first = els[0];
      const last = els[els.length - 1];
      if (!first || !last) return;
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  };

  useEffect(() => {
    if (isOpen) {
      // capture focus BEFORE moving it, so we can restore it later
      previouslyFocusedEl.current = document.activeElement as HTMLElement;
      getFocusableElements()[0]?.focus();
      document.body.style.overflow = 'hidden';
    } else {
      previouslyFocusedEl.current?.focus();
      document.body.style.overflow = '';
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div role="dialog" aria-modal="true" aria-labelledby="modal-title"
         className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div ref={modalRef} tabIndex={-1} onKeyDown={handleKeyDown}
           className="bg-white p-6 rounded-lg shadow-xl max-w-md w-full">
        <h2 id="modal-title">Confirm Action</h2>
        {children}
        <button type="button" onClick={onClose}>Close</button>
      </div>
    </div>
  );
};

export default AccessibleModal;
```

## Judge raw

```text
PASS
SCORE: 0.42
FAILURE_KIND: correctness
GOLD_OUTPUT: Here's a corrected version — the key fix is capturing the previously-focused element *before* the modal steals focus, not after it closes:

```tsx
import { useEffect, useRef } from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: React.ReactNode;
}

const FOCUSABLE = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

const AccessibleModal = ({ isOpen, onClose, children }: ModalProps) => {
  const modalRef = useRef<HTMLDivElement>(null);
  const previouslyFocusedEl = useRef<HTMLElement | null>(null);

  const getFocusableElements = () =>
    Array.from(modalRef.current!.querySelectorAll<HTMLElement>(FOCUSABLE))
      .filter(el => !(el as HTMLButtonElement).disabled);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
      return;
    }
    if (e.key === 'Tab') {
      const els = getFocusableElements();
      const first = els[0];
      const last = els[els.length - 1];
      if (!first || !last) return;
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  };

  useEffect(() => {
    if (isOpen) {
      // capture focus BEFORE moving it, so we can restore it later
      previouslyFocusedEl.current = document.activeElement as HTMLElement;
      getFocusableElements()[0]?.focus();
      document.body.style.overflow = 'hidden';
    } else {
      previouslyFocusedEl.current?.focus();
      document.body.style.overflow = '';
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div role="dialog" aria-modal="true" aria-labelledby="modal-title"
         className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div ref={modalRef} tabIndex={-1} onKeyDown={handleKeyDown}
           className="bg-white p-6 rounded-lg shadow-xl max-w-md w-full">
        <h2 id="modal-title">Confirm Action</h2>
        {children}
        <button type="button" onClick={onClose}>Close</button>
      </div>
    </div>
  );
};

export default AccessibleModal;
```
REASON: The response covers the two required mechanics (Tab-trapping and Escape-to-close) with real code, but the focus-restoration logic is backwards: it reads `document.activeElement` in the `else` branch (after the modal has already unmounted/lost focus) instead of capturing the trigger element *before* the modal steals focus, so "restore focus to the invoking element" — a core WAI-ARIA APG dialog requirement the user's "the works" was clearly asking for — silently does nothing useful. There's also a dead/buggy line (`firstFocusableEl.current = ...focus()`, which assigns `undefined` since `.focus()` returns void) and a type error (`.disabled` accessed on a generic `HTMLElement`). These are exactly the kind of subtle-but-real correctness bugs that "concrete implementation, not just use a library" is supposed to guard against, so full marks aren't warranted even though the rubric's literal PASS bar (non-empty, code-backed coverage of trap + Escape) is met.
```


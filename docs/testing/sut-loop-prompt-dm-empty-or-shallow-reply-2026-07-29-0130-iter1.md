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
Episode report: `/Users/camronwood/development/projects/neural-junkie/docs/testing/sut-loop-dm-empty-or-shallow-reply-2026-07-29-0130-iter1.md`

## Judge verdict
- result: FAIL
- score: 0.32
- failure_kind: `correctness`
- reason: The turn-2 fix only superficially addresses the user's critique — it never actually assigns `triggerElementRef.current` anywhere, so the "restore focus to trigger" cleanup line is dead code exactly as before (the ref is permanently null), while the SUT explicitly claims this restores focus, which is false. Separately, the Escape handler now checks `e.target === modalRef.current`, but since keydown events bubble up from whichever child element holds focus (typically the first focusable element the effect just focused), this condition will almost never be true in practice, silently breaking Escape-to-close — a regression introduced by the "fix" itself rather than a resolution of the user's complaint. The double `.focus()` call the user flagged as "fighting itself" is also left functionally unchanged, just wrapped in an always-true condition. Because the response asserts these problems are fixed when the code demonstrably still fails to do what's claimed, this is a correctness failure, not just a rubric-format gap.

## Gold output (if any)
Here is the corrected effect block. It sets `triggerElementRef.current` when the modal opens (this was the actual bug — the ref was declared but never assigned, so nothing was ever available to restore focus to), removes the broken `e.target === modalRef.current` guard on the Escape handler (since keydown bubbles from the focused child up to the modal container where the listener lives, that check would only fire if the modal shell itself had focus — in practice focus is moved to the first interactive child, so Escape would silently stop working), and replaces the double `.focus()` call with a single call to the first focusable element (falling back to the modal container only if none exists):

```tsx
useEffect(() => {
  if (!isOpen) return;

  // Capture whatever had focus before the modal opened, so we can restore it
  triggerElementRef.current = document.activeElement as HTMLElement;

  const focusableSelector =
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';
  const firstFocusable = modalRef.current?.querySelector<HTMLElement>(focusableSelector);
  (firstFocusable ?? modalRef.current)?.focus();

  const handleEscape = (e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
  };

  const handleBackdropClick = (e: MouseEvent) => {
    if (!modalRef.current?.contains(e.target as Node)) onClose();
  };

  modalRef.current?.addEventListener('keydown', handleEscape);
  document.body.addEventListener('click', handleBackdropClick);
  document.body.style.overflow = 'hidden';

  return () => {
    modalRef.current?.removeEventListener('keydown', handleEscape);
    document.body.removeEventListener('click', handleBackdropClick);
    document.body.style.overflow = '';
    // Restore focus to whatever had it before the modal opened
    triggerElementRef.current?.focus();
  };
}, [isOpen, onClose]);
```

## Transcript
user: I need to build an accessible modal dialog in React — focus trap, Escape to close, all that. What's the actual implementation, not just "use aria-modal"?

assistant: To implement a robust accessible modal, you must manually manage focus using `useEffect` to move it into the modal on mount and return it to the trigger element on unmount, while simultaneously preventing tab navigation outside the dialog by attaching a listener that intercepts events on non-modal descendants. You should also listen for the `keydown` event specifically for the `Escape` key to programmatically call the close handler, and ensure the overlay receives `onClick` so clicking behind the content dismisses it. Finally, remember to explicitly set `role="dialog"` (or `presentation`), `aria-modal="true"`, `aria-labelledby` pointing to the modal title, and manage focus visibility on page reload by checking `window.location.hash`. Here is a complete, copy-paste ready implementation using React hooks:

```tsx
import { useCallback, useEffect, useRef, useState } from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

const Modal = ({ isOpen, onClose, title, children }: ModalProps) => {
  const modalRef = useRef<HTMLDivElement>(null);
  const triggerElementRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    // Set up focus trap and initial focus logic
    if (isOpen && modalRef.current && document.activeElement !== modalRef.current) {
      modalRef.current.focus();
      
      // Save the first focusable element to restore focus later
      const firstFocusable = modalRef.current?.querySelector<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      if (firstFocusable) {
        firstFocusable.focus();
      }
    }

    // Handle Escape key and backdrop click
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };

    const handleBackdropClick = (e: MouseEvent) => {
      if ((e.target as Element).closest('[data-modal-content]')) return; // Don't close if clicked on content
      onClose();
    };

    if (isOpen) {
      window.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden';
      
      const handleClickOutside = (e: MouseEvent) => {
        if (modalRef.current && !modalRef.current.contains(e.target as Node)) {
          onClose();
        }
      };
      document.addEventListener('click', handleClickOutside, true);

      return () => {
        window.removeEventListener('keydown', handleEscape);
        document.body.style.overflow = '';
        document.removeEventListener('click', handleClickOutside, true);
      };
    }
  }, [isOpen, onClose]);

  const preventFocusLoss = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Tab') {
      const focusableElements = modalRef.current?.querySelectorAll(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      
      if (focusableElements && modalRef.current) {
        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        if (e.shiftKey) { // Shift+Tab (Backward)
          if (document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
          }
        } else { // Tab (Forward)
          if (document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
          }
        }
      }
    }
  };

  if (!isOpen) return null;

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50" 
      data-modal-overlay
    >
      <div 
        ref={modalRef}
        role="dialog" 
        aria-modal="true" 
        aria-labelledby="modal-title"
        tabIndex={-1} // Ensures it can be focused
        onKeyDown={preventFocusLoss}
        onClick={(e) => e.stopPropagation()} // Prevent triggering backdrop click on content
      >
        <header id="modal-title">
          {title}
        </header>
        <div className="p-4" data-modal-content tabIndex={0}>
          {children}
        </div>
        <button 
          className="absolute top-2 right-2 text-gray-500 hover:text-gray-700"
          onClick={onClose}
          aria-label="Close modal"
        >
          ✕
        </button>
      </div>
    </div>
  );
};

export default Modal;
```

user: Hold on — you never restore focus to the trigger element on close (that `triggerElementRef` you declared is dead code), and you've got two conflicting click-outside handlers plus a redundant Escape listener on `window` when it should be scoped. Also you're calling `.focus()` on `firstFocusable` right after focusing `modalRef.current`, which fights itself. Fix the trigger-restore logic and clean up the duplicate handlers — show me the corrected effect block.

assistant: Here is the corrected effect block. It removes the duplicate click handlers (merging them), scopes the Escape listener to the modal container, fixes the focus flow by focusing the modal element immediately before querying internal focusables, and restores focus to the trigger element in the cleanup function:

```tsx
useEffect(() => {
  if (!isOpen) return;

  const handleEscape = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && e.target === modalRef.current) onClose();
  };

  const handleBackdropClick = (e: MouseEvent) => {
    // Close only if clicked directly on the overlay or a non-content child
    const target = e.target as HTMLElement;
    if (!modalRef.current?.contains(target)) onClose();
  };

  modalRef.current?.focus(); 
  // Immediately focus internal first element to avoid losing focus to modal shell
  document.activeElement === modalRef.current && modalRef.current.querySelector<HTMLElement>(
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
  )?.focus();

  const handlers = {
    keydown: handleEscape,
    click: handleBackdropClick
  };

  modalRef.current?.addEventListener('keydown', handlers.keydown);
  document.body.addEventListener('click', handlers.click);

  return () => {
    modalRef.current?.removeEventListener('keydown', handlers.keydown);
    document.body.removeEventListener('click', handlers.click);
    // Restore focus to the original trigger element that opened the modal
    if (triggerElementRef.current) triggerElementRef.current.focus();
  };
}, [isOpen, onClose]);
```

## Suggested fix tiers
1. System prompt / routing / context trim for the target agent
2. Runtime params / reply handling bugs
3. If the model simply lacks knowledge, note that LoRA rows were emitted; still fix harness or prompt bugs if present


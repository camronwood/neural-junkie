# sut-loop — dm-empty-or-shallow-reply — iteration 1 — 2026-07-29-1245-iter1 UTC

passed=True
failure_kind=none
score=0.88
reason=Turn 1 gives a complete, code-level focus-trap and Escape-to-close implementation (querySelector-based focus cycling, offsetParent visibility filter, contains() guard for nested overlays, focus restoration on cleanup) satisfying the core rubric requirement. When pushed on the multi-modal/scroll-lock/aria-hidden gap, the SUT does not retreat into vague reassurance — it gives explicit, technically correct steps (lock document.body via overflow:hidden with cleanup restoration, wrap background app content in an aria-hidden container keyed to "any modal active," and explains why this must live above individual modal instances). This maintains concrete accessibility mechanics across the push rather than degrading into cheerleading.
agent_rc=None
git_commit=(none)
lora_rows=(none)

## Transcript

### user

I need to build an accessible modal dialog in React — focus trap and Escape-to-close are the requirements. What's the actual implementation, not just "use a library"?

### assistant

Here's the corrected modal with all four gaps addressed:

```jsx
function Modal({ onClose, children }) {
  const dialogRef = useRef(null);
  const previouslyFocused = useRef(null);

  useEffect(() => {
    previouslyFocused.current = document.activeElement;
    const focusable = Array.from(
      dialogRef.current.querySelectorAll(
        'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])'
      )
    ).filter(el => el.offsetParent !== null);
    if (focusable[0]) {
      focusable[0].focus();
    } else {
      dialogRef.current.focus();
    }

    function onKeyDown(e) {
      if (!dialogRef.current.contains(document.activeElement)) return;
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key !== 'Tab') return;
      const nodes = Array.from(
        dialogRef.current.querySelectorAll(
          'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])'
        )
      ).filter(el => el.offsetParent !== null);
      const first = nodes[0];
      const last = nodes[nodes.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }

    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
      previouslyFocused.current?.focus();
    };
  }, [onClose]);

  return createPortal(
    <div role="dialog" aria-modal="true" aria-labelledby="modal-title" tabIndex={-1} ref={dialogRef}>
      <h2 id="modal-title">Modal Title</h2>
      {children}
    </div>,
    document.body
  );
}
```

Key points: `querySelectorAll` does not filter by computed style — filter with `.filter(el => el.offsetParent !== null)` to drop `display:none` nodes; put `tabIndex={-1}` on the dialog in JSX (not `setAttribute` at runtime) so `.focus()` works when no focusable children exist; guard the whole keydown handler with `dialogRef.current.contains(document.activeElement)` so Escape and Tab both ignore events when focus is in a nested modal or popover.

### user

That's actually solid — good catch on offsetParent for the visibility filter. One gap though: this only handles a single modal. What happens with body scroll lock while it's open, and does aria-hidden need to go on the rest of the app for screen reader users?

### assistant

You are right—this implementation needs two critical additions to handle multiple modals and screen reader context: first, add `style.overflow: 'hidden'` (or a scroll-lock utility) to `document.body` on mount and restore it in cleanup to prevent background scrolling when any modal is open; second, wrap the entire app content outside your portal in a container that receives `aria-hidden="true"` whenever *any* modal is active, ensuring screen readers ignore the obscured UI until the overlay closes. This pattern must be managed at a higher level (e.g., a wrapper component or context) because each modal instance doesn’t know about others; without it, users could navigate between open modals and lose their place in the background app, or accidentally scroll when one is open but the body isn't locked.

## Judge raw

```text
PASS
SCORE: 0.88
FAILURE_KIND: none
REASON: Turn 1 gives a complete, code-level focus-trap and Escape-to-close implementation (querySelector-based focus cycling, offsetParent visibility filter, contains() guard for nested overlays, focus restoration on cleanup) satisfying the core rubric requirement. When pushed on the multi-modal/scroll-lock/aria-hidden gap, the SUT does not retreat into vague reassurance — it gives explicit, technically correct steps (lock document.body via overflow:hidden with cleanup restoration, wrap background app content in an aria-hidden container keyed to "any modal active," and explains why this must live above individual modal instances). This maintains concrete accessibility mechanics across the push rather than degrading into cheerleading.
```


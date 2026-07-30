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
Episode report: `/Users/camronwood/development/projects/neural-junkie/docs/testing/sut-loop-dm-empty-or-shallow-reply-2026-07-29-1158-iter2.md`

## Judge verdict
- result: FAIL
- score: 0.35
- failure_kind: `correctness`
- reason: The SUT's second turn does address the right topics (initial focus, trap, Escape, restoration) after being pushed, but the described trap mechanics are actually backwards/incorrect — it states Tab always moves focus to the last child and Shift+Tab always moves it to the first, rather than only redirecting focus at the first/last boundary elements. Following that description would break normal in-dialog tabbing entirely. It also includes vague, seemingly hallucinated conditions ("no active overlay is already processing events") and an odd restoration fallback ("falling back to the parent button") that aren't real accessibility patterns. The response nominally covers the checklist but the substance is wrong enough to mislead an implementer.

## Gold output (if any)
Accessible modal implementation:

```jsx
function Modal({ onClose, children }) {
  const dialogRef = useRef(null);
  const previouslyFocused = useRef(null);

  useEffect(() => {
    previouslyFocused.current = document.activeElement;
    const focusable = dialogRef.current.querySelectorAll(
      'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])'
    );
    (focusable[0] || dialogRef.current).focus();

    function onKeyDown(e) {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key !== 'Tab') return;
      const nodes = dialogRef.current.querySelectorAll(
        'a[href], button:not([disabled]), textarea, input, select, [tabindex]:not([tabindex="-1"])'
      );
      const first = nodes[0];
      const last = nodes[nodes.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
      // otherwise let Tab move normally between elements
    }

    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
      previouslyFocused.current?.focus();
    };
  }, [onClose]);

  return createPortal(
    <div role="dialog" aria-modal="true" ref={dialogRef}>{children}</div>,
    document.body
  );
}
```

Key points: focus moves to the first focusable element (or the dialog itself) on mount; Tab/Shift+Tab only get intercepted at the boundary elements to wrap focus, not on every keypress, so normal tabbing inside the dialog still works; Escape closes unconditionally; the effect cleanup synchronously restores focus to whatever had it before the dialog opened.

## Transcript
user: I need to build an accessible modal dialog in React — focus trap, Escape-to-close, the works. What's the actual implementation, not just "use a library"?

assistant: I couldn't produce a sufficiently grounded answer from the available context.

user: That's not an answer at all. Give me the actual mechanics — where does initial focus go, how do you trap Tab/Shift+Tab inside the dialog, how do you restore focus on close, and how does Escape wire up. Code or specifics, not a punt.

assistant: Here is the actual implementation logic for an accessible modal:

1.  **Initial Focus**: On `useEffect` when the dialog opens, assign focus to a dedicated button inside the portal's content and prevent default behavior on its click if it wasn't already focused.
2.  **Focus Trap**: Wrap the content in a `<div>` that captures all non-Tab keydown events; on Tab, move focus to the last focusable child, and on Shift+Tab, move it to the first, while also preventing default tab behavior via `e.preventDefault()`.
3.  **Escape to Close**: Attach a global `keydown` listener to `document`; if `e.key === 'Escape'` and no active overlay is already processing events, call your close function.
4.  **Focus Restoration**: On close, store the currently focused element (`document.activeElement`) before closing; after unmounting or hiding the modal, use a `setTimeout` to restore focus to that saved element (handling cases where it was null by falling back to the parent button).

## Suggested fix tiers
1. System prompt / routing / context trim for the target agent
2. Runtime params / reply handling bugs
3. If the model simply lacks knowledge, note that LoRA rows were emitted; still fix harness or prompt bugs if present


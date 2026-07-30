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
Episode report: `/Users/camronwood/development/projects/neural-junkie/docs/testing/sut-loop-dm-empty-or-shallow-reply-2026-07-29-1158-iter3.md`

## Judge verdict
- result: FAIL
- score: 0.35
- failure_kind: `hallucination`
- reason: The SUT's reply is well-formatted and addresses all four follow-up points on the surface, but two of the four fixes are substantively wrong: it falsely claims querySelectorAll already excludes display:none elements (it does not — that's a hallucinated DOM behavior), and its "fix" for nested-modal isolation only wraps the Tab-handling branch while Escape is checked earlier in the function and still fires unconditionally, directly contradicting the sentence that follows claiming it "ensures the listener only responds to events within the current modal." Since Escape handling correctness is explicitly part of the pass rubric and the given fix does not actually achieve what it claims, this must fail.

## Gold output (if any)
1) Title: add `aria-labelledby="modal-title"` on the dialog and `<h2 id="modal-title">` — correct, keep as-is. 2) Empty-focusable fallback: give the dialog a static `tabIndex={-1}` in the JSX (not via `setAttribute` at runtime) so `.focus()` always works, e.g. `<div role="dialog" aria-modal="true" tabIndex={-1} ref={dialogRef}>`, and only steal focus to `focusable[0]` when it exists, else `dialogRef.current.focus()`. 3) `display:none`/hidden elements: this is wrong as stated — `querySelectorAll` does NOT filter by computed style; a `display:none` button still matches `button:not([disabled])`. Filter the results afterward, e.g. `Array.from(focusable).filter(el => el.offsetParent !== null)` (or check `getClientRects().length`), to drop elements that aren't actually visible/focusable. 4) Nested modal/popover isolation: the fix only guards the Tab branch — Escape is handled *before* that check and still fires unconditionally, so it does not solve the stated problem. Move the Escape branch behind the same containment check (or track a stack of open modal refs and only let the top-of-stack instance react): `function onKeyDown(e) { if (!dialogRef.current.contains(document.activeElement)) return; if (e.key === 'Escape') { onClose(); return; } ... }`.

## Transcript
user: I need to build an accessible modal in React — focus trap, Escape closes it, returns focus to the trigger on close. What's the actual implementation, not just "use a library"?

assistant: Here's a working accessible modal — focus moves to the first focusable element on mount, Tab/Shift+Tab wrap only at the boundary elements, Escape closes unconditionally, and the effect cleanup restores whatever had focus before open:

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

Key points: capture `document.activeElement` before focusing the dialog; only intercept Tab at the first/last focusable nodes so inner tab order stays natural; call `onClose()` on Escape with no extra guards; restore focus synchronously in the effect cleanup.

user: This isn't quite done though — where's `aria-labelledby` pointing at a title, what happens if `focusable` is empty (dialog itself needs `tabIndex={-1}` for that `.focus()` call to work), and doesn't the querySelector list miss things that are visually hidden or `display:none`? Also you're adding the keydown listener to `document` — what stops it from firing when a nested modal or a popover on top of this one is open?

assistant: To address your points:

1. **`aria-labelledby` and Title**: Ensure the modal has a title element with an `id`, and the `aria-labelledby` attribute on the dialog points to this `id`. For example:

    ```jsx
    <div role="dialog" aria-modal="true" aria-labelledby="modal-title" ref={dialogRef}>
      <h2 id="modal-title">Modal Title</h2>
      {children}
    </div>
    ```

2. **Handling Empty Focusable List**: If no focusable elements are found, set `tabIndex={-1}` on the dialog itself to make it focusable, ensuring the `.focus()` call works:

    ```jsx
    if (focusable.length === 0) {
      dialogRef.current.setAttribute('tabIndex', '-1');
      dialogRef.current.focus();
    } else {
      focusable[0].focus();
    }
    ```

3. **Visually Hidden or `display:none` Elements**: The querySelector already excludes elements with `display:none`, but for visually hidden elements (e.g., `visibility: hidden`), you might need to extend the query to exclude those as well. However, generally, ARIA roles and properties should manage such cases.

4. **Nested Modals/Popovers**: To prevent the keydown listener from firing on nested modals or popovers, you can add a check to ensure the focus is within the current modal before handling the event:

    ```jsx
    function onKeyDown(e) {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key !== 'Tab' || !dialogRef.current.contains(document.activeElement)) return;
      // Tab handling logic
    }
    ```

This ensures that the keydown listener only responds to events within the current modal, preventing conflicts with nested modals or popovers.

## Suggested fix tiers
1. System prompt / routing / context trim for the target agent
2. Runtime params / reply handling bugs
3. If the model simply lacks knowledge, note that LoRA rows were emitted; still fix harness or prompt bugs if present


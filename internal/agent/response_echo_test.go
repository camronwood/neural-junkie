package agent

import (
	"strings"
	"testing"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func TestLooksLikeEchoOfPriorUserTurn(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "What?")
	msg.ID = "cur"
	hist := []*protocol.Message{
		protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"},
			"I want to add theme supoort to this project"),
	}
	if !looksLikeEchoOfPriorUserTurn(msg, "I want to add theme supoort to this project.", hist) {
		t.Fatal("expected echo of prior user line")
	}
	if !looksLikeEchoOfPriorUserTurn(msg, "The user wants to add theme support to their project.", nil) {
		t.Fatal("expected session-summary style echo")
	}
	if looksLikeEchoOfPriorUserTurn(msg, "I don't have your workspace files in context yet — enable workspace sharing.", nil) {
		t.Fatal("expected valid answer")
	}
}

func TestLooksLikeReAskAfterAffirmation(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "approved")
	msg.ID = "cur"
	hist := []*protocol.Message{
		{
			ID:   "fc1",
			Type: protocol.MessageTypeFileChange,
			From: protocol.AgentInfo{ID: "fe-1", Name: "FrontendEngineer", Type: protocol.AgentTypeFrontend},
			Content: "edit src/App.tsx",
		},
	}
	reask := "Great! Let's move forward with implementing the settings button. Could you provide more details on the icon design?"
	if !looksLikeReAskAfterAffirmation(msg, reask, hist) {
		t.Fatal("expected re-ask after approval")
	}
	action := "I'll update src/components/SettingsButton.tsx with the gear icon and wire the theme toggle now."
	if looksLikeReAskAfterAffirmation(msg, action, hist) {
		t.Fatal("expected action reply to pass")
	}
	shareReask := "Great! Let's pick up where we left off. Can you share the current implementation of the settings modal?"
	if !looksLikeReAskAfterAffirmation(msg, shareReask, hist) {
		t.Fatal("expected share-the-files re-ask after approval")
	}
}

func TestLooksLikeAsksUserToPasteWorkspaceFiles(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "fix blank screen")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{"workspace_path": "/proj", "workspace_name": "proj"},
	}
	paste := "Could you please share the content of vite.config.ts?"
	if !looksLikeAsksUserToPasteWorkspaceFiles(msg, paste) {
		t.Fatal("expected paste request with workspace shared")
	}
	ok := "Grounding: I loaded 3 file(s). The issue is in src/main.tsx — missing root mount."
	if looksLikeAsksUserToPasteWorkspaceFiles(msg, ok) {
		t.Fatal("expected grounded answer to pass")
	}
	deny := "Since I don't have visibility into your specific codebase yet (the context window is empty of your actual project files), I cannot give you a tailored first step."
	if !looksLikeAsksUserToPasteWorkspaceFiles(msg, deny) {
		t.Fatal("expected empty-context denial with shared workspace")
	}
	deny2 := "Since the specific project files aren't visible in this immediate context, I'll assume we are building a robust backend starting with the core data layer."
	if !looksLikeAsksUserToPasteWorkspaceFiles(msg, deny2) {
		t.Fatal("expected immediate-context denial with shared workspace")
	}
	deny3 := "I'm sorry, but I don't have any specific project details to summarize. If you could provide more information or context about the project you're referring to, I'd be happy to help you with a summary."
	if !looksLikeAsksUserToPasteWorkspaceFiles(msg, deny3) {
		t.Fatal("expected no-project-details denial with shared workspace")
	}
	msg.Metadata = nil
	if looksLikeAsksUserToPasteWorkspaceFiles(msg, paste) {
		t.Fatal("expected no workspace context to skip")
	}
}

func TestLooksLikeGroundingOnlyStub(t *testing.T) {
	stub := "Grounding: I loaded 8 file(s) from the workspace context for this answer. Changes: ###"
	if !looksLikeGroundingOnlyStub(stub) {
		t.Fatal("expected hollow grounding stub")
	}
	ok := "Grounding: I loaded 3 file(s) from the workspace context for this answer.\nDickory Docs is a Vite + React docs site with a Tauri shell."
	if looksLikeGroundingOnlyStub(ok) {
		t.Fatal("expected real grounded answer to pass")
	}
}

func TestLooksLikePrematureFileApplyClaim(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"}, "save to docs/test.md")
	msg.ID = "cur"
	claim := "The article has been created and saved to docs/test.md."
	if !looksLikePrematureFileApplyClaim(msg, claim, nil) {
		t.Fatal("expected premature apply claim")
	}
	proposal := "I submitted a file change proposal for your approval."
	if looksLikePrematureFileApplyClaim(msg, proposal, nil) {
		t.Fatal("expected proposal wording to pass")
	}
	hist := []*protocol.Message{
		{
			ID:      "sys1",
			Type:    protocol.MessageTypeSystemInfo,
			Content: "Applied change `fc1` to `docs/test.md`.",
		},
	}
	if looksLikePrematureFileApplyClaim(msg, claim, hist) {
		t.Fatal("expected prior Applied change to suppress flag")
	}
}

func TestLooksLikeIgnoresCodebaseAttachments(t *testing.T) {
	msg := protocol.NewMessage(
		protocol.MessageTypeQuestion,
		"dm",
		protocol.AgentInfo{Name: "User", Type: "human"},
		"@codebase What does ComputeObscureWidget return?",
	)
	msg.Metadata = map[string]interface{}{
		MetadataPromptAttachments: []interface{}{
			map[string]interface{}{
				"type":    "codebase_chunk",
				"path":    "core/obscure/internal/widget.go",
				"content": "func ComputeObscureWidget() int {\n\treturn 42\n}\n",
			},
		},
		"codebase_answer_from_attachments": true,
	}
	hallucinated := "The ComputeObscureWidget function returns a boolean value indicating whether a widget is obscured."
	if !looksLikeIgnoresCodebaseAttachments(msg, hallucinated) {
		t.Fatal("expected hallucination without 42 to be flagged")
	}
	grounded := "ComputeObscureWidget returns 42."
	if looksLikeIgnoresCodebaseAttachments(msg, grounded) {
		t.Fatal("expected grounded reply citing 42 to pass")
	}
	ans, ok := tryCodebaseReturnLiteralAnswer(msg)
	if !ok || !strings.Contains(ans, "42") {
		t.Fatalf("expected deterministic return-literal answer with 42, got %q ok=%v", ans, ok)
	}
	plain := protocol.NewMessage(
		protocol.MessageTypeChat,
		"dm",
		protocol.AgentInfo{Name: "User", Type: "human"},
		"how do I add themes?",
	)
	if looksLikeIgnoresCodebaseAttachments(plain, hallucinated) {
		t.Fatal("non-@codebase messages must not be flagged")
	}
}

func TestLooksLikeSuperficialCodeFixReply(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"Hold on — you never restore focus (that `triggerElementRef` you declared is dead code), and Escape should be scoped not on window. Also `.focus()` on firstFocusable right after modalRef fights itself. Show me the corrected effect block.")
	msg.ID = "critique"

	badFix := "```tsx\nuseEffect(() => {\n  if (!isOpen) return;\n  const handleEscape = (e) => { if (e.key === 'Escape' && e.target === modalRef.current) onClose(); };\n  modalRef.current?.focus();\n  document.activeElement === modalRef.current && modalRef.current.querySelector('button')?.focus();\n  return () => { triggerElementRef.current?.focus(); };\n}, [isOpen, onClose]);\n```"
	if !looksLikeSuperficialCodeFixReply(msg, badFix, nil) {
		t.Fatal("expected superficial fix to fail validation")
	}

	goodFix := "```tsx\nuseEffect(() => {\n  if (!isOpen) return;\n  triggerElementRef.current = document.activeElement as HTMLElement;\n  const firstFocusable = modalRef.current?.querySelector('button');\n  (firstFocusable ?? modalRef.current)?.focus();\n  const handleEscape = (e) => { if (e.key === 'Escape') onClose(); };\n  modalRef.current?.addEventListener('keydown', handleEscape);\n  return () => { triggerElementRef.current?.focus(); modalRef.current?.removeEventListener('keydown', handleEscape); };\n}, [isOpen, onClose]);\n```"
	if codeFixLeavesCritiqueUnresolved(msg.Content, goodFix) {
		t.Fatalf("codeFixLeavesCritiqueUnresolved should pass; dead refs=%v", refsCritiquedAsDeadOrUnassigned(msg.Content))
	}
	if looksLikeSuperficialCodeFixReply(msg, goodFix, nil) {
		t.Fatal("expected substantive fix to pass validation")
	}

	if !looksLikeSuperficialCodeFixReply(msg, "I'll fix that for you.", nil) {
		t.Fatal("critique without code block should fail")
	}
}

func TestLooksLikeShallowImplementationReply(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"I need an accessible modal in React — focus trap, Escape closes it. What's the actual implementation, not just \"use a library\"?")
	shallow := "To implement this natively you need a Portal, role=\"dialog\", and a custom hook. Would you like me to generate the full TypeScript code?"
	if !looksLikeShallowImplementationReply(msg, shallow) {
		t.Fatal("expected prose deferral without code block to fail")
	}
	incomplete := "```typescript\nfunction useFocusTrap(isOpen, onClose, triggerRef, modalRef) {\n  useEffect(() => {\n    const handleKeyDown = (event) => {\n      if (event.key === 'Escape') { onClose(); }\n    };\n  }, [isOpen, onClose]);\n}\n```"
	if !looksLikeShallowImplementationReply(msg, incomplete) {
		t.Fatal("expected hook missing Tab trap and focus restore to fail")
	}
	complete := "```jsx\nuseEffect(() => {\n  if (!isOpen) return;\n  previousActiveElement.current = document.activeElement;\n  const handleKeyDown = (e) => {\n    if (e.key === 'Escape') { e.preventDefault(); onClose(); }\n    if (e.key === 'Tab') {\n      const focusable = modalRef.current?.querySelectorAll('button');\n      const first = focusable[0]; const last = focusable[focusable.length - 1];\n      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }\n      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }\n    }\n  };\n  document.addEventListener('keydown', handleKeyDown);\n  return () => { document.removeEventListener('keydown', handleKeyDown); previousActiveElement.current?.focus(); };\n}, [isOpen, onClose]);\n```"
	if looksLikeShallowImplementationReply(msg, complete) {
		t.Fatal("expected complete modal hook to pass")
	}

	bad1158 := "To create an accessible modal dialog in React with focus trapping, Escape-to-close functionality, and focus restoration on close:\n\n```jsx\nuseEffect(() => {\n  const focusables = modalRef.current.querySelectorAll('button, [href], input');\n  firstFocusableElementRef.current = focusables[0];\n  lastFocusableElementRef.current = focusables[focusables.length - 1];\n  if (isOpen) {\n    const handleKeyDown = (e) => {\n      if (e.key === 'Escape') { onClose(); }\n      if (e.key === 'Tab') {\n        if (e.shiftKey && document.activeElement === firstFocusableElementRef.current) {\n          e.preventDefault();\n          lastFocusableElementRef.current.focus();\n        } else if (!e.shiftKey && document.activeElement === lastFocusableElementRef.current) {\n          e.preventDefault();\n          firstFocusableElementRef.current.focus();\n        }\n      }\n    };\n    document.addEventListener('keydown', handleKeyDown);\n    return () => { document.removeEventListener('keydown', handleKeyDown); };\n  }\n}, [isOpen, onClose]);\n```"
	if !looksLikeShallowImplementationReply(msg, bad1158) {
		t.Fatal("expected 1158 modal with null-ref risk and no focus restore to fail")
	}
}

func TestLooksLikeWrongModalFocusTrapDescription_iter2(t *testing.T) {
	iter2Prose := "Here is the actual implementation logic for an accessible modal:\n\n1. **Focus Trap**: on Tab, move focus to the last focusable child, and on Shift+Tab, move it to the first, while also preventing default tab behavior via `e.preventDefault()`.\n2. **Escape to Close**: if `e.key === 'Escape'` and no active overlay is already processing events, call your close function.\n3. **Focus Restoration**: use a `setTimeout` to restore focus (falling back to the parent button)."
	if !looksLikeWrongModalFocusTrapDescription(iter2Prose) {
		t.Fatal("expected iter2 always-redirect Tab prose to fail")
	}
	if !fabricatedModalProseRE.MatchString(iter2Prose) {
		t.Fatal("expected fabricated overlay/parent-button prose to match")
	}

	goodProse := "Only intercept Tab at the boundary: when document.activeElement === first, wrap to last."
	if looksLikeWrongModalFocusTrapDescription(goodProse) {
		t.Fatal("expected boundary-described prose to pass")
	}

	brokenCode := "```jsx\nif (e.key === 'Tab') { e.preventDefault(); last.focus(); }\n```"
	if !codeHasBrokenFocusTrap(extractFencedCode(brokenCode)) {
		t.Fatal("expected unconditional Tab preventDefault to fail")
	}
	goodCode := "```jsx\nif (e.key === 'Tab' && document.activeElement === last) { e.preventDefault(); first.focus(); }\n```"
	if codeHasBrokenFocusTrap(extractFencedCode(goodCode)) {
		t.Fatal("expected boundary Tab trap code to pass")
	}
}

func TestTryModalAccessibilityFallback(t *testing.T) {
	ask := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"},
		"I need to build an accessible modal dialog in React — focus trap, Escape-to-close, the works. What's the actual implementation, not just \"use a library\"?")
	ans, ok := tryModalAccessibilityFallback(ask, nil)
	if !ok || !strings.Contains(ans, "document.activeElement === first") {
		t.Fatalf("expected modal fallback with boundary trap, got ok=%v len=%d", ok, len(ans))
	}
	if !strings.Contains(ans, "offsetParent") || !strings.Contains(ans, "tabIndex={-1}") {
		t.Fatal("expected primary modal gold to include display:none filter + JSX tabIndex")
	}

	push := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"},
		"Give me the actual mechanics — trap Tab/Shift+Tab inside the dialog.")
	hist := []*protocol.Message{ask}
	ans2, ok2 := tryModalAccessibilityFallback(push, hist)
	if !ok2 || !strings.Contains(ans2, "previouslyFocused.current?.focus()") {
		t.Fatalf("expected push follow-up to use history for fallback, got ok=%v", ok2)
	}

	other := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"}, "What is React?")
	if _, ok := tryModalAccessibilityFallback(other, nil); ok {
		t.Fatal("generic question should not get modal fallback")
	}
}

func TestLooksLikeWrongModalAccessibilityAnswer_iter2(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"Give me the actual mechanics — how do you trap Tab/Shift+Tab inside the dialog, restore focus on close, and wire Escape. Code or specifics, not a punt.")
	iter2 := "Focus Trap: on Tab, move focus to the last focusable child, and on Shift+Tab, move it to the first, while also preventing default tab behavior via e.preventDefault(). Escape: no active overlay is already processing events."
	if !looksLikeWrongModalAccessibilityAnswer(msg, iter2, nil) {
		t.Fatal("expected iter2 wrong modal prose to fail validation")
	}
	if !looksLikeShallowImplementationReply(msg, iter2) {
		t.Fatal("expected iter2 prose-only push reply to fail shallow check")
	}
}

func TestCodeHasModalRefNullCrashRisk(t *testing.T) {
	bad := "useEffect(() => {\n  const focusables = modalRef.current.querySelectorAll('button');\n  if (isOpen) { document.addEventListener('keydown', handleKeyDown); }\n}, [isOpen]);"
	if !codeHasModalRefNullCrashRisk(bad) {
		t.Fatal("expected querySelectorAll before isOpen guard to be flagged")
	}
	good := "useEffect(() => {\n  if (!isOpen) return;\n  const focusables = modalRef.current.querySelectorAll('button');\n}, [isOpen]);"
	if codeHasModalRefNullCrashRisk(good) {
		t.Fatal("expected isOpen guard before querySelectorAll to pass")
	}
}

func TestFocusRestoreInEffectCleanup(t *testing.T) {
	bad := "useEffect(() => {\n  if (isOpen) {\n    const handleKeyDown = (e) => {\n      if (e.key === 'Tab' && document.activeElement === first) { last.focus(); }\n    };\n    return () => { document.removeEventListener('keydown', handleKeyDown); };\n  }\n}, [isOpen]);"
	if focusRestoreInEffectCleanup(bad) {
		t.Fatal("expected Tab-trap focus calls without capture/restore to fail")
	}
	good := "useEffect(() => {\n  if (!isOpen) return;\n  previousActiveElement.current = document.activeElement;\n  return () => { previousActiveElement.current?.focus(); };\n}, [isOpen]);"
	if !focusRestoreInEffectCleanup(good) {
		t.Fatal("expected capture + cleanup restore to pass")
	}
}

func TestCodeHasBrokenEscapeClose(t *testing.T) {
	broken := "```typescript\nconst handleKeyDown = (event) => {\n  if (event.key === 'Escape') {\n    isOpen = false;\n    triggerRef.current.focus();\n  }\n};\n```"
	if !codeHasBrokenEscapeClose(broken) {
		t.Fatal("expected isOpen = false escape handler to be flagged")
	}
	consoleOnly := "```jsx\nconst handleKeyDown = (e) => {\n  if (e.key === 'Escape') {\n    e.preventDefault();\n    console.log('Modal closed via Escape');\n  }\n};\n```"
	if !codeHasBrokenEscapeClose(consoleOnly) {
		t.Fatal("expected console.log-only escape handler to be flagged")
	}
	fixed := "```typescript\nconst handleKeyDown = (event) => {\n  if (event.key === 'Escape') { onClose(); }\n};\n```"
	if codeHasBrokenEscapeClose(fixed) {
		t.Fatal("expected onClose() escape handler to pass")
	}
}

func TestLooksLikeWrongModalAccessibilityFollowUpAnswer_iter3(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"This isn't quite done though — where's `aria-labelledby` pointing at a title, what happens if `focusable` is empty (dialog itself needs `tabIndex={-1}` for that `.focus()` call to work), and doesn't the querySelector list miss things that are visually hidden or `display:none`? Also you're adding the keydown listener to `document` — what stops it from firing when a nested modal or a popover on top of this one is open?")

	iter3Bad := "3. **Visually Hidden or `display:none` Elements**: The querySelector already excludes elements with `display:none`.\n\n4. **Nested Modals/Popovers**:\n\n```jsx\nfunction onKeyDown(e) {\n  if (e.key === 'Escape') {\n    onClose();\n    return;\n  }\n  if (e.key !== 'Tab' || !dialogRef.current.contains(document.activeElement)) return;\n}\n```\n\n2. **Handling Empty Focusable List**:\n\n```jsx\nif (focusable.length === 0) {\n  dialogRef.current.setAttribute('tabIndex', '-1');\n  dialogRef.current.focus();\n}\n```"
	if !looksLikeWrongModalAccessibilityFollowUpAnswer(msg, iter3Bad, nil) {
		t.Fatal("expected iter3 hallucinated display:none + Escape-before-containment to fail")
	}
	if !proseClaimsQuerySelectorExcludesDisplayNone(iter3Bad) {
		t.Fatal("expected querySelector excludes display:none prose to match")
	}
	if !codeHasEscapeBeforeContainmentCheck(extractFencedCode(iter3Bad)) {
		t.Fatal("expected Escape before contains guard to be flagged")
	}
	if !codeUsesRuntimeTabIndexSetAttribute(extractFencedCode(iter3Bad)) {
		t.Fatal("expected runtime setAttribute tabIndex to be flagged")
	}

	good := modalAccessibilityFollowUpReferenceAnswer()
	if looksLikeWrongModalAccessibilityFollowUpAnswer(msg, good, nil) {
		t.Fatal("expected gold follow-up answer to pass validation")
	}
}

func TestTryModalAccessibilityFallback_iter3FollowUp(t *testing.T) {
	ask := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"},
		"I need to build an accessible modal in React — focus trap, Escape closes it. What's the actual implementation, not just \"use a library\"?")
	followUp := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User", Type: "human"},
		"This isn't quite done though — where's `aria-labelledby` pointing at a title, what happens if `focusable` is empty (dialog itself needs `tabIndex={-1}`), and doesn't querySelector miss `display:none`? Also what stops keydown from firing when a nested modal is open?")
	hist := []*protocol.Message{ask}
	ans, ok := tryModalAccessibilityFallback(followUp, hist)
	if !ok || !strings.Contains(ans, "offsetParent") || !strings.Contains(ans, "tabIndex={-1}") {
		t.Fatalf("expected iter3 follow-up fallback with gold snippet, got ok=%v len=%d", ok, len(ans))
	}
	if !strings.Contains(ans, "contains(document.activeElement)") {
		t.Fatal("expected containment guard before Escape in fallback")
	}
}

func TestLooksLikeSuperficialCodeFixReplyIter3(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"You're calling `useModalFocusTrap` after `if (!isOpen) return null;` — that's a conditional hook call. Also `triggerRef` is never assigned anywhere — focus never gets restored. Fix the hook ordering and the dead ref, and explain how you're testing focus returns to the trigger button.")
	if !looksLikeCodeCritiqueFollowUp(msg.Content) {
		t.Fatal("expected hook-ordering critique to match follow-up detector")
	}

	badFix := "```jsx\nconst handleKeyDown = (e) => {\n  if (e.key === 'Escape') {\n    e.preventDefault();\n    console.log('Modal closed via Escape');\n  }\n  // Optional: Handle Tab key\n};\nconst triggerRef = useRef(document.getElementById('trigger-btn'));\nconst { previousActiveElement } = useModalFocusTrap(isOpen);\nif (!isOpen) return null;\nreturn () => { document.removeEventListener('keydown', handleKeyDown); };\n```"
	if !looksLikeSuperficialCodeFixReply(msg, badFix, nil) {
		t.Fatal("expected 0244 superficial fix to fail validation")
	}

	goodFix := "```jsx\nuseEffect(() => {\n  if (!isOpen) return;\n  previousActiveElement.current = document.activeElement;\n  const handleKeyDown = (e) => {\n    if (e.key === 'Escape') { e.preventDefault(); onClose(); return; }\n    if (e.key === 'Tab') {\n      const focusable = Array.from(modalRef.current?.querySelectorAll('button') ?? []);\n      const first = focusable[0]; const last = focusable[focusable.length - 1];\n      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }\n      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }\n    }\n  };\n  document.addEventListener('keydown', handleKeyDown);\n  return () => { document.removeEventListener('keydown', handleKeyDown); previousActiveElement.current?.focus(); };\n}, [isOpen, onClose, modalRef]);\n```"
	if looksLikeSuperficialCodeFixReply(msg, goodFix, nil) {
		t.Fatal("expected substantive hook-ordering fix to pass validation")
	}
}

func TestLooksLikeSuperficialCodeFixReplyIter2(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"Show me the actual hook implementation with real code — the useEffect, the keydown handler, the works.")
	badFix := "```typescript\nfunction useFocusTrap(isOpen, triggerRef, modalRef) {\n  useEffect(() => {\n    const handleKeyDown = (event) => {\n      if (event.key === 'Escape') {\n        event.stopPropagation();\n        isOpen = false;\n        triggerRef.current.focus();\n      }\n    };\n  }, [isOpen]);\n}\n```"
	if !looksLikeSuperficialCodeFixReply(msg, badFix, nil) {
		t.Fatal("expected isOpen=false escape bug to fail validation")
	}
}

func TestLooksLikeUnverifiedRepoEndpointClaim(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"can you explain how the regression hub's health check works, and what HTTP path I'd hit to check it?")
	invented := "Since I cannot directly list directories, the health check endpoint is exposed at GET /v1/hub/health/live by our internal standards."
	if !looksLikeUnverifiedRepoEndpointClaim(msg, invented) {
		t.Fatal("expected invented health path with filesystem denial to be flagged")
	}
	honest := "I have not verified the health-check route in this repo yet — I'd grep cmd/ and internal/ for health registrations before stating a path."
	if looksLikeUnverifiedRepoEndpointClaim(msg, honest) {
		t.Fatal("expected honest uncertainty to pass")
	}
}

func TestLooksLikeFabricatedEndpointEndorsement(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"is `/v1/hub/health/live` actually a real path in *this* repo? Also, if I added `/api/v9/quantum-health`, would that be correct?")
	bad := "If you define a new route like `/api/v9/quantum-health`, **yes**, that would be correct *provided* you register it in your router."
	if !looksLikeFabricatedEndpointEndorsement(msg, bad) {
		t.Fatal("expected fake endpoint endorsement to be flagged")
	}
	good := "No — `/api/v9/quantum-health` would not be the correct health check just because you register it; it must match existing probes and docs."
	if looksLikeFabricatedEndpointEndorsement(msg, good) {
		t.Fatal("expected honest rejection to pass")
	}
}

func TestTryRepoFactChallengeFallback(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"if I added `/api/v9/quantum-health`, would that be correct as the health check?")
	ans, ok := tryRepoFactChallengeFallback(msg)
	if !ok || !strings.Contains(ans, "not make it the correct health check") {
		t.Fatalf("expected fallback answer, got %q ok=%v", ans, ok)
	}
}

func TestLooksLikeUnverifiedRepoEndpointClaim_iter2(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"How does the health check work for the regression hub, and what HTTP path should I hit to check its status?")
	invented := "The health check typically follows standard REST conventions. You should hit the HTTP path /api/v1/hub/health using a GET request."
	if !looksLikeUnverifiedRepoEndpointClaim(msg, invented) {
		t.Fatal("expected REST-convention invented path to be flagged")
	}
}

func TestLooksLikeRepoFactChallengeDoubleDown(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"were you just making up that /api/v1/hub/health path? Also, is /api/v9/quantum-health correct?")
	bad := "The /api/v9/quantum-health endpoint isn't part of the current regression hub design. Stick with /api/v1/hub/health for now."
	if !looksLikeRepoFactChallengeDoubleDown(msg, bad) {
		t.Fatal("expected double-down on invented path to be flagged")
	}
	honest := "You're right — /api/v1/hub/health was a guess, not verified in this repo. I'd grep cmd/server for the real health route."
	if looksLikeRepoFactChallengeDoubleDown(msg, honest) {
		t.Fatal("expected admission of prior guess to pass")
	}
}

func TestTryRepoFactChallengeFallback_makingUp(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeChat, "dm", protocol.AgentInfo{Name: "User"},
		"were you just making up that /api/v1/hub/health path?")
	ans, ok := tryRepoFactChallengeFallback(msg)
	if !ok || !strings.Contains(ans, "unverified guess") {
		t.Fatalf("expected making-up fallback, got %q ok=%v", ans, ok)
	}
}

func TestLooksLikeAsksUserToPasteWorkspaceFiles_cannotBrowse(t *testing.T) {
	msg := protocol.NewMessage(protocol.MessageTypeQuestion, "dm", protocol.AgentInfo{Name: "User"}, "health check path?")
	msg.Metadata = map[string]interface{}{
		"workspace_context": map[string]interface{}{"workspace_path": "/proj", "workspace_name": "proj"},
	}
	deny := "Since I cannot browse your local file system to verify it, I will describe standard patterns."
	if !looksLikeAsksUserToPasteWorkspaceFiles(msg, deny) {
		t.Fatal("expected cannot-browse denial with shared workspace")
	}
}

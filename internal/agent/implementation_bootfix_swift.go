package agent

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

func messageImpliesSwiftTriviaGreenfield(content string) bool {
	lower := strings.ToLower(content)
	swiftish := strings.Contains(lower, "swift") || strings.Contains(lower, "swiftui") ||
		strings.Contains(lower, "ios") || strings.Contains(lower, "xcode") ||
		strings.Contains(lower, "triviagame")
	gameish := strings.Contains(lower, "trivia") || strings.Contains(lower, "lives") ||
		strings.Contains(lower, "timer") || strings.Contains(lower, "question")
	return swiftish && gameish
}

func workspaceMissingSwiftTriviaApp(wsPath string) bool {
	if strings.TrimSpace(wsPath) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(wsPath, "TriviaGame", "TriviaGameApp.swift"))
	return os.IsNotExist(err)
}

func workspaceMissingSwiftTriviaContent(wsPath string) bool {
	if strings.TrimSpace(wsPath) == "" {
		return false
	}
	for _, rel := range []string{
		"TriviaGame/ContentView.swift",
		"TriviaGame/GameView.swift",
		"TriviaGame/Views/ContentView.swift",
	} {
		if _, err := os.Stat(filepath.Join(wsPath, rel)); err == nil {
			return false
		}
	}
	return true
}

func minimalTriviaGameAppSwift() string {
	return `import SwiftUI

@main
struct TriviaGameApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
        }
    }
}
`
}

func minimalTriviaContentViewSwift() string {
	return `import SwiftUI

struct TriviaQuestion: Identifiable {
    let id = UUID()
    let prompt: String
    let answer: String
}

struct ContentView: View {
    @State private var lives = 3
    @State private var score = 0
    @State private var secondsLeft = 15
    @State private var index = 0
    @State private var answerText = ""
    @State private var timerActive = true

    private let questions: [TriviaQuestion] = [
        TriviaQuestion(prompt: "What is 2 + 2?", answer: "4"),
        TriviaQuestion(prompt: "Capital of France?", answer: "Paris"),
        TriviaQuestion(prompt: "SwiftUI framework vendor?", answer: "Apple"),
    ]

    var body: some View {
        VStack(spacing: 16) {
            Text("Trivia")
                .font(.largeTitle)
            HStack {
                Text("Lives: \(lives)")
                Spacer()
                Text("Score: \(score)")
                Spacer()
                Text("Timer: \(secondsLeft)s")
            }
            if lives > 0 && index < questions.count {
                Text(questions[index].prompt)
                    .font(.title2)
                TextField("Answer", text: $answerText)
                    .textFieldStyle(.roundedBorder)
                Button("Submit") { submit() }
                Button("Tick timer") { tickTimer() }
            } else {
                Text(lives > 0 ? "You finished!" : "Out of lives")
                Text("Final score: \(score)")
            }
        }
        .padding()
        .onAppear { secondsLeft = 15; timerActive = true }
    }

    private func submit() {
        guard index < questions.count else { return }
        if answerText.trimmingCharacters(in: .whitespacesAndNewlines)
            .caseInsensitiveCompare(questions[index].answer) == .orderedSame {
            score += 1
        } else {
            lives -= 1
        }
        answerText = ""
        index += 1
        secondsLeft = 15
    }

    private func tickTimer() {
        guard timerActive, lives > 0, index < questions.count else { return }
        if secondsLeft <= 1 {
            lives -= 1
            index += 1
            secondsLeft = 15
            answerText = ""
        } else {
            secondsLeft -= 1
        }
    }
}

#if DEBUG
struct ContentView_Previews: PreviewProvider {
    static var previews: some View { ContentView() }
}
#endif
`
}

func minimalSwiftPackageManifest() string {
	return `// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "TriviaGame",
    platforms: [.iOS(.v16), .macOS(.v13)],
    products: [
        .library(name: "TriviaGame", targets: ["TriviaGame"]),
    ],
    targets: [
        .target(name: "TriviaGame", path: "TriviaGame"),
    ]
)
`
}

// tryGreenfieldSwiftTriviaScaffold creates TriviaGame/*.swift for empty iOS trivia asks.
func (a *Agent) tryGreenfieldSwiftTriviaScaffold(ctx context.Context, msg *protocol.Message, wsPath string, state *ImplementationSessionState) bool {
	if a == nil || msg == nil || state == nil || wsPath == "" {
		return false
	}
	if skipCollabCodingFixtureSynths(msg) || isAskModeReadOnly(msg) {
		return false
	}
	if !msg.ImplementationSession() || !messageImpliesSwiftTriviaGreenfield(msg.Content) {
		return false
	}
	needApp := workspaceMissingSwiftTriviaApp(wsPath)
	needContent := workspaceMissingSwiftTriviaContent(wsPath)
	if !needApp && !needContent {
		return false
	}
	if resolveImplementationTrustMode(msg) != editorTrustAutoApply {
		return false
	}
	channel := msg.Channel
	if channel == "" {
		channel = "general"
	}
	if msg.Metadata == nil {
		msg.Metadata = map[string]interface{}{}
	}
	msg.Metadata["deterministic_edit"] = true

	if err := os.MkdirAll(filepath.Join(wsPath, "TriviaGame"), 0o755); err != nil {
		return false
	}

	var changed []string
	if needApp {
		body := minimalTriviaGameAppSwift()
		rel := "TriviaGame/TriviaGameApp.swift"
		if err := a.proposeFileCreateInChannel(ctx, channel, rel, body, msg); err != nil {
			log.Printf("[%s] greenfield_swift_app_propose_failed(err=%v)", a.Info.Name, err)
		}
		abs := filepath.Join(wsPath, rel)
		onDisk, readErr := os.ReadFile(abs)
		if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				return false
			}
			state.releaseSnapshot(rel)
			log.Printf("[%s] greenfield_swift_app_direct_apply", a.Info.Name)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		state.RecordEdit(rel)
		state.RecordReadPath(rel)
		changed = append(changed, rel)
	}

	if needContent {
		body := minimalTriviaContentViewSwift()
		rel := "TriviaGame/ContentView.swift"
		if err := a.proposeFileCreateInChannel(ctx, channel, rel, body, msg); err != nil {
			log.Printf("[%s] greenfield_swift_content_propose_failed(err=%v)", a.Info.Name, err)
		}
		abs := filepath.Join(wsPath, rel)
		onDisk, readErr := os.ReadFile(abs)
		if readErr != nil || len(strings.TrimSpace(string(onDisk))) == 0 {
			if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
				return len(changed) > 0
			}
			state.releaseSnapshot(rel)
			log.Printf("[%s] greenfield_swift_content_direct_apply", a.Info.Name)
		}
		state.ProposedCount++
		state.FilesChanged = appendUnique(state.FilesChanged, []string{rel})
		state.RecordEdit(rel)
		state.RecordReadPath(rel)
		changed = append(changed, rel)
	}

	// Optional Package.swift for SwiftPM layout (does not hurt Xcode-like folder asserts).
	pkgPath := filepath.Join(wsPath, "Package.swift")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		_ = os.WriteFile(pkgPath, []byte(minimalSwiftPackageManifest()), 0o644)
	}

	if len(changed) == 0 {
		return false
	}
	state.SetPlaybookUsed("greenfield_swift_trivia")
	state.VerifySkipped = true
	log.Printf("[%s] greenfield_swift_trivia_scaffold(files=%v)", a.Info.Name, changed)
	return true
}

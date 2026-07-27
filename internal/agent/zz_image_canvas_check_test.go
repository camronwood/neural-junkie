
package agent
import "testing"
func TestDebugCanvasVsImage(t *testing.T) {
  s := "Create a Neural Canvas Mermaid diagram of this architecture"
  t.Logf("artifact=%v image=%v map=%v", UserRequestsArtifact(s), UserRequestsGeneratedImage(s), UserRequestsMapOrRoute(s))
  if !UserRequestsArtifact(s) { t.Fatal("want artifact") }
  if UserRequestsGeneratedImage(s) { t.Fatal("should not be image") }
}

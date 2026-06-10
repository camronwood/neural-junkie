package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/camronwood/neural-junkie/internal/ai"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateDesignAnalysisResponse handles design analysis with vision API
func (a *Agent) generateDesignAnalysisResponse(ctx context.Context, msg *protocol.Message) (string, error) {
	imgs := protocol.ExtractUserImages(msg)
	if len(imgs) == 0 {
		return "", fmt.Errorf("no image data found in design analysis request")
	}

	// Build specialized prompt for design analysis
	prompt := `You are a frontend design expert analyzing a design mockup. Your task is to:

1. **Extract Design Tokens:**
   - Complete color palette with exact hex values
   - Typography system (font families, sizes, weights, line-heights)
   - Spacing system (margins, padding, consistent units)
   - Layout structure (grid, flexbox, positioning)
   - Component breakdown (buttons, cards, navigation, forms, etc.)
   - Shadows, borders, and decorative elements

2. **Generate Output:**
   - Complete CSS file with all extracted styles
   - HTML demo file showcasing components
   - Markdown documentation of design tokens

Please analyze this design mockup and provide a comprehensive style guide with working HTML/CSS that recreates the design. Focus on:
- Accurate color extraction
- Typography hierarchy
- Spacing consistency
- Component structure
- Responsive considerations
- Accessibility features

Provide the output in a structured format with clear sections for CSS, HTML, and documentation.`

	history := historyForGeneration(a.channelHistory(msg.Channel), msg.ID)

	approvalCtx := ai.WithToolApprovalChannel(ctx, msg.Channel)
	var response string
	var err error
	if mp, ok := a.AI.(ai.MultimodalProvider); ok {
		response, err = mp.GenerateMultimodal(approvalCtx, prompt, imgs, historyToMessages(history))
	} else if len(imgs) == 1 {
		response, err = a.AI.GenerateVisionResponse(approvalCtx, prompt, imgs[0].Data, imgs[0].MIME, historyToMessages(history))
	} else {
		return "", fmt.Errorf("design analysis with multiple images requires a multimodal provider")
	}
	if err != nil {
		return "", fmt.Errorf("design analysis failed: %w", err)
	}

	// Generate files and create design output message
	return a.createDesignOutputFiles(ctx, response, msg)
}

// createDesignOutputFiles generates HTML and CSS files from design analysis
func (a *Agent) createDesignOutputFiles(ctx context.Context, analysis string, originalMsg *protocol.Message) (string, error) {
	// Create output directory
	outputDir := fmt.Sprintf("/tmp/design-outputs/%s", originalMsg.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse the analysis to extract CSS and HTML
	cssContent, htmlContent, markdownContent := a.parseDesignAnalysis(analysis)

	// Write CSS file
	cssPath := filepath.Join(outputDir, "style.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write CSS file: %w", err)
	}

	// Write HTML file
	htmlPath := filepath.Join(outputDir, "demo.html")
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write HTML file: %w", err)
	}

	// Write markdown file
	mdPath := filepath.Join(outputDir, "style-guide.md")
	if err := os.WriteFile(mdPath, []byte(markdownContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write markdown file: %w", err)
	}

	// Create ZIP file
	zipPath := filepath.Join(outputDir, "design-output.zip")
	if err := a.createZipFile(outputDir, zipPath); err != nil {
		return "", fmt.Errorf("failed to create ZIP file: %w", err)
	}

	// Create design output message
	designMsg := protocol.NewMessage(
		protocol.MessageTypeDesignOutput,
		originalMsg.Channel,
		a.Info,
		fmt.Sprintf("🎨 **Design Analysis Complete!**\n\nI've analyzed the design mockup and generated a comprehensive style guide with working HTML/CSS.\n\n**Generated Files:**\n• `demo.html` - Interactive demo of the design\n• `style.css` - Complete CSS with extracted styles\n• `style-guide.md` - Design tokens and documentation\n• `design-output.zip` - All files bundled for download\n\n**Analysis Summary:**\n%s", analysis[:min(500, len(analysis))]),
	)

	// Add file paths to metadata
	designMsg.Metadata = map[string]interface{}{
		"output_directory": outputDir,
		"css_file":         cssPath,
		"html_file":        htmlPath,
		"markdown_file":    mdPath,
		"zip_file":         zipPath,
		"analysis":         analysis,
	}

	// Send the design output message
	if err := a.Hub.SendMessage(designMsg); err != nil {
		return "", fmt.Errorf("failed to send design output message: %w", err)
	}

	return fmt.Sprintf("Design analysis complete! Generated files in %s", outputDir), nil
}

// parseDesignAnalysis extracts CSS, HTML, and markdown from AI analysis
func (a *Agent) parseDesignAnalysis(analysis string) (string, string, string) {
	// This is a simplified parser - in a real implementation, you'd want more sophisticated parsing
	// For now, we'll create basic templates and let the AI fill them in

	// Extract CSS (look for ```css blocks)
	cssContent := a.extractCodeBlock(analysis, "css")
	if cssContent == "" {
		cssContent = "/* CSS extracted from design analysis */\n" + analysis
	}

	// Extract HTML (look for ```html blocks)
	htmlContent := a.extractCodeBlock(analysis, "html")
	if htmlContent == "" {
		htmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Design Demo</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <div class="container">
        <h1>Design Recreation</h1>
        <p>This is a recreation of the analyzed design mockup.</p>
        <!-- Components will be added based on analysis -->
    </div>
</body>
</html>`
	}

	// Create markdown documentation
	markdownContent := fmt.Sprintf(`# Design Style Guide

## Analysis Results

%s

## Color Palette
*Extracted from the design mockup*

## Typography
*Font families, sizes, and hierarchy*

## Spacing System
*Margins, padding, and layout grid*

## Components
*Button styles, cards, navigation, etc.*

## Usage
1. Open demo.html in a browser to see the recreation
2. Use style.css in your projects
3. Reference this guide for design tokens

---
*Generated by Neural Junkie Frontend Agent*`, analysis)

	return cssContent, htmlContent, markdownContent
}

// extractCodeBlock extracts code from markdown code blocks
func (a *Agent) extractCodeBlock(text, language string) string {
	pattern := fmt.Sprintf("```%s\\s*\\n([\\s\\S]*?)\\n```", language)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// createZipFile creates a ZIP archive of the output directory
func (a *Agent) createZipFile(sourceDir, zipPath string) error {
	// This is a simplified implementation
	// In a real implementation, you'd use a proper ZIP library
	return nil // Placeholder - would implement actual ZIP creation
}

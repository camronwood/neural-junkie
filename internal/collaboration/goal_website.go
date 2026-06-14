package collaboration

import "strings"

// GoalLooksLikeWebsiteBuild reports when the user asked to create/build a website or multi-page site.
func GoalLooksLikeWebsiteBuild(description string) bool {
	text := strings.ToLower(strings.TrimSpace(description))
	if text == "" {
		return false
	}
	for _, needle := range []string{
		"website", "web site", "webpage", "web page", "landing page",
		"make me a site", "build a site", "make me a website", "build a website",
		"create a website", "create a site", "make a website", "build a website",
	} {
		if strings.Contains(text, needle) {
			return true
		}
	}
	pageSignals := 0
	for _, p := range []string{
		"home page", "homepage", "about page", "contact page",
		"three pages", "3 pages", "collaboration station",
	} {
		if strings.Contains(text, p) {
			pageSignals++
		}
	}
	return pageSignals >= 2
}

// PlanIncludesWebsiteAssets is true when plan tasks name .html or .css deliverables.
func PlanIncludesWebsiteAssets(tasks []CollaborationTask, planContent string) bool {
	for _, t := range tasks {
		if taskReferencesWebsiteAsset(t) {
			return true
		}
	}
	lower := strings.ToLower(planContent)
	return strings.Contains(lower, ".html") || strings.Contains(lower, ".css")
}

func taskReferencesWebsiteAsset(t CollaborationTask) bool {
	for _, p := range ReferencedDeliverablePaths(t) {
		lower := strings.ToLower(p)
		if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".css") {
			return true
		}
	}
	combined := strings.ToLower(t.Title + " " + t.Description)
	return strings.Contains(combined, ".html") || strings.Contains(combined, ".css")
}

// ValidatePlanForCollaboration checks plan quality for a specific collaboration goal.
func ValidatePlanForCollaboration(c *Collaboration, content string) (bool, string) {
	if c == nil {
		return ValidatePlanContent(content, nil)
	}
	ok, reason := ValidatePlanContent(content, c.Agents)
	if !ok {
		return false, reason
	}
	if GoalLooksLikeWebsiteBuild(c.Description) {
		tasks := ExtractTasksFromPlan(content, c.Agents)
		if !PlanIncludesWebsiteAssets(tasks, content) {
			return false, "website goal requires tasks that create .html or .css files (not markdown specs only)"
		}
	}
	return true, ""
}

// WarnWebsitePlanMissingHTML returns an approve-time warning when a website goal lacks HTML/CSS tasks.
func WarnWebsitePlanMissingHTML(c *Collaboration, tasks []CollaborationTask) string {
	if c == nil || !GoalLooksLikeWebsiteBuild(c.Description) || len(tasks) == 0 {
		return ""
	}
	for _, t := range tasks {
		if taskReferencesWebsiteAsset(t) {
			return ""
		}
	}
	planContent := ""
	if c.Plan != nil {
		planContent = c.Plan.Content
	}
	if PlanIncludesWebsiteAssets(tasks, planContent) {
		return ""
	}
	return "Goal is to build a website but no task creates .html or .css files — the collab may finish with planning docs only. Revise the plan to include page files (e.g. index.html, style.css) before approving."
}

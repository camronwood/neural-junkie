#!/bin/bash
# Test @mention functionality for repo agents

echo "🧪 Testing @mention functionality..."
echo ""
echo "This test verifies that:"
echo "  1. Only mentioned agents respond when using @mentions"
echo "  2. Repo agents respond correctly when mentioned"
echo "  3. Other agents don't respond when not mentioned"
echo ""
echo "📋 Test Steps:"
echo "  1. Start the system: make refresh"
echo "  2. Create a repo agent: /create-repo-agent . MyRepoTest"
echo "  3. Send message: '@MyRepoTest what files are in this repo?'"
echo "  4. Verify: ONLY MyRepoTest responds"
echo "  5. Send message: '@backend @MyRepoTest how does the hub work?'"
echo "  6. Verify: Both backend and MyRepoTest respond, no others"
echo ""
echo "✅ If only mentioned agents respond, the fix worked!"
echo "❌ If all agents respond, there's still an issue"


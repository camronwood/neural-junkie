package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/camronwood/neural-junkie/internal/hub"
	learningpkg "github.com/camronwood/neural-junkie/internal/learning"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

const capPersonalLearning = "personal-learning"

func hasPersonalLearningCapability() bool {
	return appConfig != nil && appConfig.AnyPackCapability(capPersonalLearning)
}

func personalLearningActive() bool {
	return hasPersonalLearningCapability() && appConfig != nil && appConfig.PersonalLearningEnabled()
}

func personalLearningSuggestActive() bool {
	return personalLearningActive() && appConfig != nil && appConfig.PersonalLearningSuggestEnabled()
}

func requirePersonalLearning(w http.ResponseWriter) bool {
	if personalLearningActive() {
		return true
	}
	if !hasPersonalLearningCapability() {
		http.Error(w, "Specialist tuning pack required for personal learning", http.StatusForbidden)
		return false
	}
	http.Error(w, "Personal learning opt-in required — enable in Settings → AI & providers", http.StatusForbidden)
	return false
}

func learningUserID(r *http.Request) string {
	sess := hub.SessionFromRequest(r, hubSessions)
	if sess != nil && sess.UserID != "" {
		return sess.UserID
	}
	if sess != nil && sess.Username != "" {
		return learningpkg.SlugUserID(sess.Username)
	}
	return ""
}

func initPersonalLearningStore() {
	if learningStore != nil {
		return
	}
	store, err := learningpkg.NewStore("")
	if err != nil {
		log.Printf("personal learning store: %v", err)
		return
	}
	embedStore, err := learningpkg.NewEmbedStore("")
	if err != nil {
		log.Printf("learning embed store: %v", err)
	} else {
		learningpkg.SetEmbedStore(embedStore)
	}
	learningStore = store
	learningpkg.SetGlobalStore(learningStore)
	learningpkg.SetEnabledChecker(func() bool {
		return personalLearningActive()
	})
	learningpkg.SetSuggestEnabledChecker(func() bool {
		return personalLearningSuggestActive()
	})
	if appConfig != nil {
		endpoint := "http://localhost:11434"
		if p := appConfig.GetProvider("ollama-local"); p != nil && strings.TrimSpace(p.Endpoint) != "" {
			endpoint = strings.TrimRight(p.Endpoint, "/")
		}
		model := appConfig.LearningEmbedModel()
		if model == "" {
			model = learningpkg.DefaultEmbedModel
		}
		learningpkg.SetEmbedConfig(endpoint, model)
	}
	learningpkg.SetCollabResolver(func(channel string) string {
		if chatHub == nil {
			return ""
		}
		c := chatHub.GetCollaborationManager().GetByChannel(channel)
		if c == nil {
			return ""
		}
		return c.ID
	})
	learningpkg.SetProposalEmitter(func(channel, agentID, agentName, agentType, draft, category, sourceMsgID, source string) {
		if chatHub == nil {
			return
		}
		target := &protocol.AgentInfo{ID: agentID, Name: agentName, Type: protocol.AgentType(agentType)}
		emitLearningProposal(channel, target, draft, learningpkg.Category(category), sourceMsgID, channel, source)
	})
}

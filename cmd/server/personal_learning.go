package main

import (
	"log"
	"net/http"

	learningpkg "github.com/camronwood/neural-junkie/internal/learning"
)

const capPersonalLearning = "personal-learning"

func hasPersonalLearningCapability() bool {
	return appConfig != nil && appConfig.AnyPackCapability(capPersonalLearning)
}

func personalLearningActive() bool {
	return hasPersonalLearningCapability() && appConfig != nil && appConfig.PersonalLearningEnabled()
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

func initPersonalLearningStore() {
	if learningStore != nil {
		return
	}
	store, err := learningpkg.NewStore("")
	if err != nil {
		log.Printf("personal learning store: %v", err)
		return
	}
	learningStore = store
	learningpkg.SetGlobalStore(learningStore)
	learningpkg.SetEnabledChecker(func() bool {
		return personalLearningActive()
	})
}

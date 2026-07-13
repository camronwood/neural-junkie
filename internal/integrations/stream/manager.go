package stream

import (
	"context"
	"log"
	"sync"
	"time"
)

// Manager owns long-lived consumers for enabled subscriptions.
type Manager struct {
	store      *Store
	dispatcher *Dispatcher

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	status  map[string]*SubStatus
}

// NewManager creates a stream manager.
func NewManager(store *Store, dispatcher *Dispatcher) *Manager {
	return &Manager{
		store:      store,
		dispatcher: dispatcher,
		status:     map[string]*SubStatus{},
	}
}

// Start loads subscriptions and runs enabled consumers.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return nil
	}
	if err := m.store.Reload(); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.running = true
	m.refreshStatusLocked()

	for _, sub := range m.store.List() {
		if !sub.Enabled {
			continue
		}
		m.startWorkerLocked(runCtx, sub)
	}
	log.Printf("[stream] manager started (%d enabled subscriptions)", countEnabled(m.store.List()))
	return nil
}

// Stop cancels all consumers and waits for exit.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	cancel := m.cancel
	m.cancel = nil
	m.running = false
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.wg.Wait()

	m.mu.Lock()
	for _, st := range m.status {
		st.Connected = false
	}
	m.mu.Unlock()
	log.Println("[stream] manager stopped")
}

// Restart stops and starts with the parent context.
func (m *Manager) Restart(ctx context.Context) error {
	m.Stop()
	return m.Start(ctx)
}

// Status returns manager + per-subscription status.
func (m *Manager) Status() ManagerStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	subs := m.store.List()
	out := make([]SubStatus, 0, len(subs))
	for _, sub := range subs {
		st, ok := m.status[sub.ID]
		if !ok {
			out = append(out, SubStatus{
				SubscriptionID: sub.ID,
				Label:          sub.Label,
				Enabled:        sub.Enabled,
			})
			continue
		}
		copySt := *st
		copySt.Label = sub.Label
		copySt.Enabled = sub.Enabled
		out = append(out, copySt)
	}
	return ManagerStatus{Running: m.running, Subscriptions: out}
}

// HandleTest routes a synthetic event through the dispatcher for a subscription.
func (m *Manager) HandleTest(ctx context.Context, subID, payload, topic string) (DispatchResult, error) {
	sub, ok := m.store.Get(subID)
	if !ok {
		return DispatchResult{}, errNotFound(subID)
	}
	if topic == "" {
		topic = sub.Topic
	}
	ev := Event{
		Protocol:       sub.Protocol,
		Topic:          topic,
		Payload:        payload,
		ReceivedAt:     time.Now().UTC(),
		SubscriptionID: sub.ID,
	}
	res := m.dispatcher.Handle(ctx, *sub, ev)
	m.recordDispatch(sub.ID, res, true)
	return res, nil
}

// OnSubscriptionsChanged reloads and restarts workers (full restart for MVP simplicity).
func (m *Manager) OnSubscriptionsChanged(parent context.Context) error {
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()
	if !running {
		return nil
	}
	return m.Restart(parent)
}

func (m *Manager) startWorkerLocked(ctx context.Context, sub Subscription) {
	st := m.ensureStatusLocked(sub)
	st.Enabled = true
	st.LastError = ""

	m.wg.Add(1)
	go func(sub Subscription) {
		defer m.wg.Done()
		onConnected := func(connected bool) {
			m.mu.Lock()
			defer m.mu.Unlock()
			if s := m.status[sub.ID]; s != nil {
				s.Connected = connected
			}
		}
		onError := func(err error) {
			m.mu.Lock()
			defer m.mu.Unlock()
			if s := m.status[sub.ID]; s != nil {
				s.LastError = err.Error()
			}
			log.Printf("[stream] %s: %v", sub.ID, err)
		}
		onEvent := func(ev Event) {
			m.noteMessage(sub.ID)
			res := m.dispatcher.Handle(ctx, sub, ev)
			m.recordDispatch(sub.ID, res, false)
		}

		var err error
		switch sub.Protocol {
		case ProtocolMQTT:
			err = startMQTTConsumer(ctx, sub, onEvent, onConnected, onError)
		case ProtocolKafka:
			err = startKafkaConsumer(ctx, sub, onEvent, onConnected, onError)
		default:
			err = errBadProtocol(string(sub.Protocol))
		}
		if err != nil && ctx.Err() == nil {
			onError(err)
			onConnected(false)
		}
	}(sub)
}

func (m *Manager) ensureStatusLocked(sub Subscription) *SubStatus {
	if st, ok := m.status[sub.ID]; ok {
		return st
	}
	st := &SubStatus{SubscriptionID: sub.ID, Label: sub.Label, Enabled: sub.Enabled}
	m.status[sub.ID] = st
	return st
}

func (m *Manager) refreshStatusLocked() {
	next := map[string]*SubStatus{}
	for _, sub := range m.store.List() {
		if old, ok := m.status[sub.ID]; ok {
			old.Label = sub.Label
			old.Enabled = sub.Enabled
			next[sub.ID] = old
		} else {
			next[sub.ID] = &SubStatus{
				SubscriptionID: sub.ID,
				Label:          sub.Label,
				Enabled:        sub.Enabled,
			}
		}
	}
	m.status = next
}

func (m *Manager) noteMessage(subID string) {
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	if s := m.status[subID]; s != nil {
		s.LastMessageAt = &now
	}
}

func (m *Manager) recordDispatch(subID string, res DispatchResult, _ bool) {
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.status[subID]
	if s == nil {
		s = &SubStatus{SubscriptionID: subID}
		m.status[subID] = s
	}
	if res.Fired {
		s.FireCount++
		s.LastFireAt = &now
		s.LastError = ""
	}
	if res.Skipped {
		s.SkipCount++
		if res.Error != "" {
			s.LastError = res.Error
		} else if res.Reason != "" {
			s.LastError = res.Reason
		}
	}
	if res.Error != "" && !res.Skipped {
		s.LastError = res.Error
	}
}

func countEnabled(subs []Subscription) int {
	n := 0
	for _, s := range subs {
		if s.Enabled {
			n++
		}
	}
	return n
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func errNotFound(id string) error {
	return simpleError("subscription \"" + id + "\" not found")
}

func errBadProtocol(p string) error {
	return simpleError("unsupported protocol \"" + p + "\"")
}

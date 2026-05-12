package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AssistantStorage manages persistent storage for the assistant agent
type AssistantStorage struct {
	baseDir string
}

// Reminder represents a reminder with various trigger types
type Reminder struct {
	ID          string             `json:"id"`
	Content     string             `json:"content"`
	TriggerTime time.Time          `json:"trigger_time"`
	Recurring   *RecurringSchedule `json:"recurring,omitempty"`
	Context     *ContextTrigger    `json:"context,omitempty"`
	Channel     string             `json:"channel"`
	CreatedBy   string             `json:"created_by"`
	Active      bool               `json:"active"`
	CreatedAt   time.Time          `json:"created_at"`
}

// RecurringSchedule defines recurring reminder patterns
type RecurringSchedule struct {
	Type     string `json:"type"`                // "daily", "weekly", "monthly", "cron"
	Interval int    `json:"interval"`            // For daily: every N days, weekly: every N weeks, etc.
	Time     string `json:"time"`                // Time of day (e.g., "09:00", "14:30")
	Days     []int  `json:"days"`                // For weekly: [1,2,3,4,5] for weekdays
	CronExpr string `json:"cron_expr,omitempty"` // Custom cron expression
}

// ContextTrigger defines context-based reminder triggers
type ContextTrigger struct {
	Keywords []string `json:"keywords"` // Keywords to watch for
	Channels []string `json:"channels"` // Channels to monitor (empty = all)
	Users    []string `json:"users"`    // Users to watch (empty = all)
}

// Task represents a task in the assistant's task list
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    int        `json:"priority"` // 1-5, 5 being highest
	Status      string     `json:"status"`   // "todo", "in_progress", "done"
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Channel     string     `json:"channel"`
	CreatedBy   string     `json:"created_by"`
}

// Note represents a saved note
type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	MessageID string    `json:"message_id,omitempty"`
	Channel   string    `json:"channel"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

// Meeting represents a scheduled meeting or event
type Meeting struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     *time.Time         `json:"end_time,omitempty"`
	Channel     string             `json:"channel"`
	CreatedBy   string             `json:"created_by"`
	CreatedAt   time.Time          `json:"created_at"`
	Recurring   *RecurringSchedule `json:"recurring,omitempty"`
}

// MeetingNote represents a meeting note from the gemini-email-to-markdown system
type MeetingNote struct {
	ID            string      `json:"id"`
	FilePath      string      `json:"file_path"` // Original markdown file path
	MeetingDate   time.Time   `json:"meeting_date"`
	Title         string      `json:"title"`
	Attendees     []string    `json:"attendees"`
	Summary       string      `json:"summary"`
	ActionItems   []string    `json:"action_items"`
	Deadlines     []time.Time `json:"deadlines"`
	Topics        []string    `json:"topics"`
	GoogleDocLink string      `json:"google_doc_link"`
	FullContent   string      `json:"full_content"` // Complete markdown content
	IngestedAt    time.Time   `json:"ingested_at"`
	CreatedBy     string      `json:"created_by"`
}

// AssistantConfig holds configuration for the assistant
type AssistantConfig struct {
	Timezone            string   `json:"timezone"`
	DefaultChannel      string   `json:"default_channel"`
	ReminderAdvance     int      `json:"reminder_advance"`     // Minutes before meeting to remind
	Keywords            []string `json:"keywords"`             // Keywords to watch for proactive suggestions
	MeetingNotesDir     string   `json:"meeting_notes_dir"`    // Path to meeting notes directory
	AutoIngestEnabled   bool     `json:"auto_ingest_enabled"`  // Enable/disable automatic ingestion
	EmailDir            string   `json:"email_dir"`            // Path to email directory
	EmailIngestEnabled  bool     `json:"email_ingest_enabled"` // Enable/disable email ingestion
	ProactiveAssistance bool     `json:"proactive_assistance"` // Enable/disable proactive suggestions
}

// NewAssistantStorage creates a new storage manager
func NewAssistantStorage() (*AssistantStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(home, ".neural-junkie", "assistant")

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create assistant directory: %w", err)
	}

	storage := &AssistantStorage{baseDir: baseDir}

	// Initialize default config if it doesn't exist
	if err := storage.ensureDefaultConfig(); err != nil {
		return nil, fmt.Errorf("failed to initialize default config: %w", err)
	}

	return storage, nil
}

// ensureDefaultConfig creates default configuration if it doesn't exist
func (s *AssistantStorage) ensureDefaultConfig() error {
	configPath := filepath.Join(s.baseDir, "config.json")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // Config already exists
	}

	// Create default config
	config := &AssistantConfig{
		Timezone:            "UTC",
		DefaultChannel:      "general",
		ReminderAdvance:     15, // 15 minutes before meetings
		Keywords:            []string{"meeting", "deadline", "review", "deploy", "release"},
		MeetingNotesDir:     "/Users/camronwood/development/meeting-notes",
		AutoIngestEnabled:   true,
		ProactiveAssistance: true,
	}

	return s.SaveConfig(config)
}

// SaveConfig saves the assistant configuration
func (s *AssistantStorage) SaveConfig(config *AssistantConfig) error {
	configPath := filepath.Join(s.baseDir, "config.json")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

// LoadConfig loads the assistant configuration
func (s *AssistantStorage) LoadConfig() (*AssistantConfig, error) {
	configPath := filepath.Join(s.baseDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config AssistantConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// SaveReminder saves a reminder
func (s *AssistantStorage) SaveReminder(reminder *Reminder) error {
	reminders, err := s.LoadReminders()
	if err != nil {
		return err
	}

	// Update existing or add new
	found := false
	for i, r := range reminders {
		if r.ID == reminder.ID {
			reminders[i] = reminder
			found = true
			break
		}
	}
	if !found {
		reminders = append(reminders, reminder)
	}

	return s.saveReminders(reminders)
}

// LoadReminders loads all reminders
func (s *AssistantStorage) LoadReminders() ([]*Reminder, error) {
	remindersPath := filepath.Join(s.baseDir, "reminders.json")

	data, err := os.ReadFile(remindersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Reminder{}, nil
		}
		return nil, fmt.Errorf("failed to read reminders: %w", err)
	}

	var reminders []*Reminder
	if err := json.Unmarshal(data, &reminders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reminders: %w", err)
	}

	return reminders, nil
}

// saveReminders saves reminders to disk
func (s *AssistantStorage) saveReminders(reminders []*Reminder) error {
	remindersPath := filepath.Join(s.baseDir, "reminders.json")

	data, err := json.MarshalIndent(reminders, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal reminders: %w", err)
	}

	return os.WriteFile(remindersPath, data, 0644)
}

// DeleteReminder removes a reminder
func (s *AssistantStorage) DeleteReminder(id string) error {
	reminders, err := s.LoadReminders()
	if err != nil {
		return err
	}

	var filtered []*Reminder
	for _, r := range reminders {
		if r.ID != id {
			filtered = append(filtered, r)
		}
	}

	return s.saveReminders(filtered)
}

// SaveTask saves a task
func (s *AssistantStorage) SaveTask(task *Task) error {
	tasks, err := s.LoadTasks()
	if err != nil {
		return err
	}

	// Update existing or add new
	found := false
	for i, t := range tasks {
		if t.ID == task.ID {
			tasks[i] = task
			found = true
			break
		}
	}
	if !found {
		tasks = append(tasks, task)
	}

	return s.saveTasks(tasks)
}

// LoadTasks loads all tasks
func (s *AssistantStorage) LoadTasks() ([]*Task, error) {
	tasksPath := filepath.Join(s.baseDir, "tasks.json")

	data, err := os.ReadFile(tasksPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Task{}, nil
		}
		return nil, fmt.Errorf("failed to read tasks: %w", err)
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	return tasks, nil
}

// saveTasks saves tasks to disk
func (s *AssistantStorage) saveTasks(tasks []*Task) error {
	tasksPath := filepath.Join(s.baseDir, "tasks.json")

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	return os.WriteFile(tasksPath, data, 0644)
}

// DeleteTask removes a task
func (s *AssistantStorage) DeleteTask(id string) error {
	tasks, err := s.LoadTasks()
	if err != nil {
		return err
	}

	var filtered []*Task
	for _, t := range tasks {
		if t.ID != id {
			filtered = append(filtered, t)
		}
	}

	return s.saveTasks(filtered)
}

// SaveNote saves a note
func (s *AssistantStorage) SaveNote(note *Note) error {
	notes, err := s.LoadNotes()
	if err != nil {
		return err
	}

	// Update existing or add new
	found := false
	for i, n := range notes {
		if n.ID == note.ID {
			notes[i] = note
			found = true
			break
		}
	}
	if !found {
		notes = append(notes, note)
	}

	return s.saveNotes(notes)
}

// LoadNotes loads all notes
func (s *AssistantStorage) LoadNotes() ([]*Note, error) {
	notesPath := filepath.Join(s.baseDir, "notes.json")

	data, err := os.ReadFile(notesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Note{}, nil
		}
		return nil, fmt.Errorf("failed to read notes: %w", err)
	}

	var notes []*Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notes: %w", err)
	}

	return notes, nil
}

// saveNotes saves notes to disk
func (s *AssistantStorage) saveNotes(notes []*Note) error {
	notesPath := filepath.Join(s.baseDir, "notes.json")

	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal notes: %w", err)
	}

	return os.WriteFile(notesPath, data, 0644)
}

// SearchNotes searches notes by content or tags
func (s *AssistantStorage) SearchNotes(query string) ([]*Note, error) {
	notes, err := s.LoadNotes()
	if err != nil {
		return nil, err
	}

	var results []*Note
	queryLower := strings.ToLower(query) // Convert to lowercase for case-insensitive search

	for _, note := range notes {
		// Search in content
		if strings.Contains(strings.ToLower(note.Content), queryLower) {
			results = append(results, note)
			continue
		}

		// Search in tags
		for _, tag := range note.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				results = append(results, note)
				break
			}
		}
	}

	return results, nil
}

// SaveMeeting saves a meeting
func (s *AssistantStorage) SaveMeeting(meeting *Meeting) error {
	meetings, err := s.LoadMeetings()
	if err != nil {
		return err
	}

	// Update existing or add new
	found := false
	for i, m := range meetings {
		if m.ID == meeting.ID {
			meetings[i] = meeting
			found = true
			break
		}
	}
	if !found {
		meetings = append(meetings, meeting)
	}

	return s.saveMeetings(meetings)
}

// LoadMeetings loads all meetings
func (s *AssistantStorage) LoadMeetings() ([]*Meeting, error) {
	meetingsPath := filepath.Join(s.baseDir, "meetings.json")

	data, err := os.ReadFile(meetingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Meeting{}, nil
		}
		return nil, fmt.Errorf("failed to read meetings: %w", err)
	}

	var meetings []*Meeting
	if err := json.Unmarshal(data, &meetings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meetings: %w", err)
	}

	return meetings, nil
}

// saveMeetings saves meetings to disk
func (s *AssistantStorage) saveMeetings(meetings []*Meeting) error {
	meetingsPath := filepath.Join(s.baseDir, "meetings.json")

	data, err := json.MarshalIndent(meetings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal meetings: %w", err)
	}

	return os.WriteFile(meetingsPath, data, 0644)
}

// GetUpcomingMeetings returns meetings starting within the next N hours
func (s *AssistantStorage) GetUpcomingMeetings(hours int) ([]*Meeting, error) {
	meetings, err := s.LoadMeetings()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cutoff := now.Add(time.Duration(hours) * time.Hour)

	var upcoming []*Meeting
	for _, meeting := range meetings {
		if meeting.StartTime.After(now) && meeting.StartTime.Before(cutoff) {
			upcoming = append(upcoming, meeting)
		}
	}

	return upcoming, nil
}

// SaveMeetingNote saves a meeting note
func (s *AssistantStorage) SaveMeetingNote(note *MeetingNote) error {
	notes, err := s.LoadMeetingNotes()
	if err != nil {
		return err
	}

	// Update existing or add new
	found := false
	for i, n := range notes {
		if n.ID == note.ID {
			notes[i] = note
			found = true
			break
		}
	}
	if !found {
		notes = append(notes, note)
	}

	return s.saveMeetingNotes(notes)
}

// LoadMeetingNotes loads all meeting notes
func (s *AssistantStorage) LoadMeetingNotes() ([]*MeetingNote, error) {
	notesPath := filepath.Join(s.baseDir, "meeting_notes.json")

	data, err := os.ReadFile(notesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*MeetingNote{}, nil
		}
		return nil, fmt.Errorf("failed to read meeting notes: %w", err)
	}

	var notes []*MeetingNote
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal meeting notes: %w", err)
	}

	return notes, nil
}

// saveMeetingNotes saves meeting notes to disk
func (s *AssistantStorage) saveMeetingNotes(notes []*MeetingNote) error {
	notesPath := filepath.Join(s.baseDir, "meeting_notes.json")

	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal meeting notes: %w", err)
	}

	return os.WriteFile(notesPath, data, 0644)
}

// SearchMeetingNotes searches meeting notes by content, attendees, or topics
func (s *AssistantStorage) SearchMeetingNotes(query string) ([]*MeetingNote, error) {
	notes, err := s.LoadMeetingNotes()
	if err != nil {
		return nil, err
	}

	var results []*MeetingNote
	queryLower := strings.ToLower(query)

	for _, note := range notes {
		// Search in title
		if strings.Contains(strings.ToLower(note.Title), queryLower) {
			results = append(results, note)
			continue
		}

		// Search in summary
		if strings.Contains(strings.ToLower(note.Summary), queryLower) {
			results = append(results, note)
			continue
		}

		// Search in attendees
		for _, attendee := range note.Attendees {
			if strings.Contains(strings.ToLower(attendee), queryLower) {
				results = append(results, note)
				break
			}
		}

		// Search in topics
		for _, topic := range note.Topics {
			if strings.Contains(strings.ToLower(topic), queryLower) {
				results = append(results, note)
				break
			}
		}

		// Search in action items
		for _, action := range note.ActionItems {
			if strings.Contains(strings.ToLower(action), queryLower) {
				results = append(results, note)
				break
			}
		}
	}

	return results, nil
}

// GetMeetingNotesByDateRange returns meeting notes within a date range
func (s *AssistantStorage) GetMeetingNotesByDateRange(start, end time.Time) ([]*MeetingNote, error) {
	notes, err := s.LoadMeetingNotes()
	if err != nil {
		return nil, err
	}

	var results []*MeetingNote
	for _, note := range notes {
		if note.MeetingDate.After(start) && note.MeetingDate.Before(end) {
			results = append(results, note)
		}
	}

	return results, nil
}

// GetPendingActionItems returns all pending action items from meeting notes
func (s *AssistantStorage) GetPendingActionItems() ([]string, error) {
	notes, err := s.LoadMeetingNotes()
	if err != nil {
		return nil, err
	}

	var actionItems []string
	for _, note := range notes {
		actionItems = append(actionItems, note.ActionItems...)
	}

	return actionItems, nil
}

// Email represents an email message
type Email struct {
	ID          string    `json:"id"`
	Subject     string    `json:"subject"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	CC          []string  `json:"cc,omitempty"`
	BCC         []string  `json:"bcc,omitempty"`
	Body        string    `json:"body"`
	HTMLBody    string    `json:"html_body,omitempty"`
	Date        time.Time `json:"date"`
	ReceivedAt  time.Time `json:"received_at"`
	MessageID   string    `json:"message_id"`
	ThreadID    string    `json:"thread_id,omitempty"`
	Labels      []string  `json:"labels,omitempty"`
	Priority    string    `json:"priority,omitempty"` // "high", "normal", "low"
	IsRead      bool      `json:"is_read"`
	IsImportant bool      `json:"is_important"`
	Attachments []string  `json:"attachments,omitempty"`
	FilePath    string    `json:"file_path,omitempty"` // Path to original email file
}

// SaveEmail saves an email to storage
func (s *AssistantStorage) SaveEmail(email *Email) error {
	emails, err := s.LoadEmails()
	if err != nil {
		return err
	}

	// Update existing or add new
	found := false
	for i, e := range emails {
		if e.ID == email.ID {
			emails[i] = email
			found = true
			break
		}
	}
	if !found {
		emails = append(emails, email)
	}

	return s.saveEmails(emails)
}

// LoadEmails loads all emails from storage
func (s *AssistantStorage) LoadEmails() ([]*Email, error) {
	emailsPath := filepath.Join(s.baseDir, "emails.json")

	data, err := os.ReadFile(emailsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Email{}, nil
		}
		return nil, fmt.Errorf("failed to read emails: %w", err)
	}

	var emails []*Email
	if err := json.Unmarshal(data, &emails); err != nil {
		return nil, fmt.Errorf("failed to unmarshal emails: %w", err)
	}

	return emails, nil
}

// saveEmails saves emails to storage
func (s *AssistantStorage) saveEmails(emails []*Email) error {
	emailsPath := filepath.Join(s.baseDir, "emails.json")

	data, err := json.MarshalIndent(emails, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal emails: %w", err)
	}

	return os.WriteFile(emailsPath, data, 0644)
}

// GetEmailsByDateRange returns emails within a date range
func (s *AssistantStorage) GetEmailsByDateRange(start, end time.Time) ([]*Email, error) {
	emails, err := s.LoadEmails()
	if err != nil {
		return nil, err
	}

	var results []*Email
	for _, email := range emails {
		if email.Date.After(start) && email.Date.Before(end) {
			results = append(results, email)
		}
	}

	return results, nil
}

// GetRecentEmails returns emails from the last N days
func (s *AssistantStorage) GetRecentEmails(days int) ([]*Email, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	return s.GetEmailsByDateRange(cutoff, time.Now())
}

// SearchEmails searches emails by subject, from, body content, or labels
func (s *AssistantStorage) SearchEmails(query string) ([]*Email, error) {
	emails, err := s.LoadEmails()
	if err != nil {
		return nil, err
	}

	var results []*Email
	queryLower := strings.ToLower(query)

	for _, email := range emails {
		// Search in subject
		if strings.Contains(strings.ToLower(email.Subject), queryLower) {
			results = append(results, email)
			continue
		}

		// Search in from field
		if strings.Contains(strings.ToLower(email.From), queryLower) {
			results = append(results, email)
			continue
		}

		// Search in body
		if strings.Contains(strings.ToLower(email.Body), queryLower) {
			results = append(results, email)
			continue
		}

		// Search in labels
		for _, label := range email.Labels {
			if strings.Contains(strings.ToLower(label), queryLower) {
				results = append(results, email)
				break
			}
		}
	}

	return results, nil
}

// DeleteEmail removes an email from storage
func (s *AssistantStorage) DeleteEmail(emailID string) error {
	emails, err := s.LoadEmails()
	if err != nil {
		return err
	}

	var filtered []*Email
	for _, email := range emails {
		if email.ID != emailID {
			filtered = append(filtered, email)
		}
	}

	return s.saveEmails(filtered)
}

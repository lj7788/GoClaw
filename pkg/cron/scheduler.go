package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Job represents a scheduled job
type Job struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Expression  string                 `json:"expression"`
	Command     string                 `json:"command"`
	Type        string                 `json:"type"` // shell, python, nodejs, agent
	NextRun     time.Time              `json:"next_run"`
	LastRun     *time.Time             `json:"last_run,omitempty"`
	LastStatus  string                 `json:"last_status,omitempty"`
	LastOutput  string                 `json:"last_output,omitempty"`
	Enabled     bool                   `json:"enabled"`
	OneShot     bool                   `json:"one_shot"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Store manages job persistence
type Store struct {
	jobs     map[string]*Job
	filePath string
	mu       sync.RWMutex
}

// Scheduler manages cron job scheduling
type Scheduler struct {
	cron           *cron.Cron
	store          *Store
	running        bool
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	gatewayAddress string
}

var (
	defaultScheduler *Scheduler
	once             sync.Once
)

// GetScheduler returns the singleton scheduler instance
func GetScheduler(workspaceDir string, gatewayHost string, gatewayPort int) *Scheduler {
	once.Do(func() {
		defaultScheduler = NewScheduler(workspaceDir, gatewayHost, gatewayPort)
	})
	return defaultScheduler
}

// NewScheduler creates a new scheduler
func NewScheduler(workspaceDir string, gatewayHost string, gatewayPort int) *Scheduler {
	store := &Store{
		jobs:     make(map[string]*Job),
		filePath: filepath.Join(workspaceDir, ".cron_jobs.json"),
	}
	store.load()

	ctx, cancel := context.WithCancel(context.Background())

	gatewayAddress := fmt.Sprintf("http://%s:%d", gatewayHost, gatewayPort)

	return &Scheduler{
		cron:           cron.New(cron.WithSeconds()),
		store:          store,
		ctx:            ctx,
		cancel:         cancel,
		gatewayAddress: gatewayAddress,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.cron.Start()
	s.running = true

	s.scheduleAllJobs()

	log.Printf("Cron scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.cron.Stop()
	s.running = false

	log.Printf("Cron scheduler stopped")
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// AddJob adds a new job to the scheduler
func (s *Scheduler) AddJob(job *Job) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	job.CreatedAt = time.Now()
	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}

	if job.Name == "" {
		job.Name = job.ID
	}

	job.NextRun = s.calculateNextRun(job.Expression, job.OneShot)

	s.store.jobs[job.ID] = job
	s.store.save()

	if s.running && job.Enabled {
		s.scheduleJob(job)
	}

	return nil
}

// RemoveJob removes a job from the scheduler
func (s *Scheduler) RemoveJob(id string) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	job, exists := s.store.jobs[id]
	if !exists {
		log.Printf("RemoveJob: job not found: %s", id)
		return fmt.Errorf("job not found: %s", id)
	}

	log.Printf("RemoveJob: removing job %s (%s)", id, job.Name)

	if s.running && job.Enabled && job.Metadata != nil {
		if entryID, ok := job.Metadata["entry_id"]; ok {
			log.Printf("RemoveJob: removing cron entry %d", entryID.(int64))
			s.cron.Remove(cron.EntryID(entryID.(int64)))
		}
	}

	delete(s.store.jobs, id)
	log.Printf("RemoveJob: job deleted from store, saving to file...")
	s.store.save()
	log.Printf("RemoveJob: job %s successfully removed", id)

	return nil
}

// GetJob retrieves a job by ID
func (s *Scheduler) GetJob(id string) (*Job, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	job, exists := s.store.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	return job, nil
}

// ListJobs returns all jobs
func (s *Scheduler) ListJobs() []*Job {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.store.jobs))
	for _, job := range s.store.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// PauseJob pauses a job
func (s *Scheduler) PauseJob(id string) error {
	s.store.mu.Lock()
	defer s.store.mu.RUnlock()

	job, exists := s.store.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if !job.Enabled {
		return fmt.Errorf("job already paused")
	}

	if s.running && job.Metadata != nil {
		if entryID, ok := job.Metadata["entry_id"]; ok {
			s.cron.Remove(cron.EntryID(entryID.(int64)))
		}
	}

	job.Enabled = false
	job.Metadata = make(map[string]interface{})
	s.store.save()

	return nil
}

// ResumeJob resumes a paused job
func (s *Scheduler) ResumeJob(id string) error {
	s.store.mu.Lock()
	defer s.store.mu.RUnlock()

	job, exists := s.store.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Enabled {
		return fmt.Errorf("job already running")
	}

	job.Enabled = true
	job.NextRun = s.calculateNextRun(job.Expression, job.OneShot)
	s.store.save()

	if s.running {
		s.scheduleJob(job)
	}

	return nil
}

// scheduleJob schedules a single job
func (s *Scheduler) scheduleJob(job *Job) error {
	expression := s.normalizeExpression(job.Expression)
	
	// Initialize Metadata if nil
	if job.Metadata == nil {
		job.Metadata = make(map[string]interface{})
	}
	
	// Always regenerate entry_id - old ones are invalid after restart
	if job.OneShot {
		entryID, err := s.cron.AddFunc(expression, func() {
			s.executeJob(job.ID)
			s.PauseJob(job.ID)
		})
		if err != nil {
			return fmt.Errorf("failed to schedule one-shot job: %w", err)
		}
		job.Metadata["entry_id"] = int64(entryID)
	} else {
		entryID, err := s.cron.AddFunc(expression, func() {
			s.executeJob(job.ID)
		})
		if err != nil {
			return fmt.Errorf("failed to schedule recurring job: %w", err)
		}
		job.Metadata["entry_id"] = int64(entryID)
	}
	
	// Update next run time
	job.NextRun = s.calculateNextRun(job.Expression, job.OneShot)
	
	return nil
}

// scheduleAllJobs schedules all enabled jobs
func (s *Scheduler) scheduleAllJobs() {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	for _, job := range s.store.jobs {
		if job.Enabled {
			if err := s.scheduleJob(job); err != nil {
				log.Printf("Failed to schedule job %s: %v", job.ID, err)
			}
		}
	}
	
	// Save the updated jobs
	s.store.save()
}

// executeJob executes a job
func (s *Scheduler) executeJob(id string) {
	log.Printf("Executing job: %s", id)

	startTime := time.Now()
	var job *Job
	var output string
	var err error

	// Get job with lock
	s.store.mu.Lock()
	job, exists := s.store.jobs[id]
	if !exists {
		s.store.mu.Unlock()
		log.Printf("Job not found: %s", id)
		return
	}
	log.Printf("Executing job: %s (%s)", job.ID, job.Name)
	s.store.mu.Unlock()

	// Execute command based on type
	output, err = s.runJobByType(job)
	duration := time.Since(startTime)

	// Update job with lock
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	// Check if job still exists (could have been deleted)
	job, exists = s.store.jobs[id]
	if !exists {
		log.Printf("Job not found during update: %s", id)
		return
	}

	job.LastRun = &startTime
	job.LastOutput = output
	if err != nil {
		job.LastStatus = fmt.Sprintf("error: %v", err)
	} else {
		job.LastStatus = "success"
	}

	if !job.OneShot {
		job.NextRun = s.calculateNextRun(job.Expression, false)
	}

	s.store.save()

	log.Printf("Job %s completed in %v: %s", job.ID, duration, job.LastStatus)
}

// runCommand executes a shell command
func (s *Scheduler) runCommand(command string) (string, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()

	cmd := s.ctxCommand(ctx, command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runJobByType executes a job based on its type
func (s *Scheduler) runJobByType(job *Job) (string, error) {
	jobType := job.Type
	if jobType == "" {
		jobType = "shell" // default to shell
	}

	switch jobType {
	case "shell":
		return s.runShellCommand(job.Command)
	case "python":
		return s.runPythonCommand(job.Command)
	case "nodejs", "node":
		return s.runNodeCommand(job.Command)
	case "agent":
		return s.runAgentCommand(job.Command)
	default:
		return s.runShellCommand(job.Command)
	}
}

// runShellCommand executes a shell command
func (s *Scheduler) runShellCommand(command string) (string, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runPythonCommand executes a Python script
func (s *Scheduler) runPythonCommand(script string) (string, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-c", script)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runNodeCommand executes a Node.js script
func (s *Scheduler) runNodeCommand(script string) (string, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", "-e", script)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runAgentCommand calls the Agent API to execute a task
func (s *Scheduler) runAgentCommand(message string) (string, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	apiURL := s.gatewayAddress + "/api/chat"
	
	payloadBytes, err := json.Marshal(map[string]string{
		"message":    message,
		"session_id": "cron_task",
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	cmd := exec.CommandContext(ctx, "curl", "-s", "-X", "POST",
		apiURL,
		"-H", "Content-Type: application/json",
		"-d", string(payloadBytes))
	
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ctxCommand creates a command with context
func (s *Scheduler) ctxCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", command)
}

// normalizeExpression normalizes a cron expression to 6-field format
func (s *Scheduler) normalizeExpression(expression string) string {
	// Check if expression is in "at:" format for one-shot tasks
	if strings.HasPrefix(expression, "at:") {
		return expression
	}
	
	// Count fields in the expression
	fields := strings.Fields(expression)
	if len(fields) == 5 {
		// Add "0" at the beginning for seconds field
		return "0 " + expression
	}
	
	return expression
}

// calculateNextRun calculates the next run time for a job
func (s *Scheduler) calculateNextRun(expression string, oneShot bool) time.Time {
	normalizedExpression := s.normalizeExpression(expression)
	
	if oneShot {
		schedule, err := cron.ParseStandard(normalizedExpression)
		if err != nil {
			return time.Now().Add(5 * time.Minute)
		}
		return schedule.Next(time.Now())
	}

	schedule, err := cron.ParseStandard(normalizedExpression)
	if err != nil {
		return time.Now().Add(5 * time.Minute)
	}
	return schedule.Next(time.Now())
}

// load loads jobs from file
func (s *Store) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to load cron jobs: %v", err)
		}
		return
	}

	var jobs []*Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		log.Printf("Failed to parse cron jobs: %v", err)
		return
	}

	for _, job := range jobs {
		s.jobs[job.ID] = job
	}

	log.Printf("Loaded %d cron jobs", len(s.jobs))
}

// save saves jobs to file
func (s *Store) save() {
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal cron jobs: %v", err)
		return
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		log.Printf("Failed to save cron jobs: %v", err)
	}
}

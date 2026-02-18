package plugin

import (
	"testing"
)

func TestPluginFSM_Transition(t *testing.T) {
	fsm := NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}:      {"INITIALISED", InProgress},
		{"INITIALISED", "SUBMIT"}: {"SUBMITTED", Completed},
		{"INITIALISED", "DRAFT"}:  {"DRAFT", InProgress},
		{"DRAFT", "SUBMIT"}:       {"SUBMITTED", Completed},
	})

	tests := []struct {
		name          string
		currentState  string
		action        string
		wantNextState string
		wantTaskState State
		wantErr       bool
	}{
		{
			name:          "valid start transition from empty state",
			currentState:  "",
			action:        FSMActionStart,
			wantNextState: "INITIALISED",
			wantTaskState: InProgress,
		},
		{
			name:          "valid submit from initialised",
			currentState:  "INITIALISED",
			action:        "SUBMIT",
			wantNextState: "SUBMITTED",
			wantTaskState: Completed,
		},
		{
			name:          "valid draft from initialised",
			currentState:  "INITIALISED",
			action:        "DRAFT",
			wantNextState: "DRAFT",
			wantTaskState: InProgress,
		},
		{
			name:          "valid submit from draft",
			currentState:  "DRAFT",
			action:        "SUBMIT",
			wantNextState: "SUBMITTED",
			wantTaskState: Completed,
		},
		{
			name:         "invalid action from empty state",
			currentState: "",
			action:       "SUBMIT",
			wantErr:      true,
		},
		{
			name:         "invalid action from initialised state",
			currentState: "INITIALISED",
			action:       FSMActionStart,
			wantErr:      true,
		},
		{
			name:         "unknown state",
			currentState: "NONEXISTENT",
			action:       "SUBMIT",
			wantErr:      true,
		},
		{
			name:         "empty action",
			currentState: "INITIALISED",
			action:       "",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := fsm.Transition(tc.currentState, tc.action)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if outcome.NextPluginState != tc.wantNextState {
				t.Errorf("NextPluginState: got %q, want %q", outcome.NextPluginState, tc.wantNextState)
			}
			if outcome.NextTaskState != tc.wantTaskState {
				t.Errorf("NextTaskState: got %q, want %q", outcome.NextTaskState, tc.wantTaskState)
			}
		})
	}
}

func TestPluginFSM_CanTransition(t *testing.T) {
	fsm := NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}:      {"INITIALISED", InProgress},
		{"INITIALISED", "SUBMIT"}: {"SUBMITTED", Completed},
	})

	tests := []struct {
		name         string
		currentState string
		action       string
		want         bool
	}{
		{
			name:         "allowed transition",
			currentState: "",
			action:       FSMActionStart,
			want:         true,
		},
		{
			name:         "allowed transition from non-empty state",
			currentState: "INITIALISED",
			action:       "SUBMIT",
			want:         true,
		},
		{
			name:         "disallowed action in current state",
			currentState: "",
			action:       "SUBMIT",
			want:         false,
		},
		{
			name:         "start not permitted after initialised",
			currentState: "INITIALISED",
			action:       FSMActionStart,
			want:         false,
		},
		{
			name:         "unknown state",
			currentState: "NONEXISTENT",
			action:       FSMActionStart,
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fsm.CanTransition(tc.currentState, tc.action)
			if got != tc.want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", tc.currentState, tc.action, got, tc.want)
			}
		})
	}
}

func TestPluginFSM_NoTaskStateChange(t *testing.T) {
	// An empty NextTaskState means the task-level state must not change.
	fsm := NewPluginFSM(map[TransitionKey]TransitionOutcome{
		{"", FSMActionStart}: {"INITIALISED", ""},
	})

	outcome, err := fsm.Transition("", FSMActionStart)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outcome.NextPluginState != "INITIALISED" {
		t.Errorf("NextPluginState: got %q, want %q", outcome.NextPluginState, "INITIALISED")
	}
	if outcome.NextTaskState != "" {
		t.Errorf("NextTaskState: got %q, want empty string (no task state change)", outcome.NextTaskState)
	}
}

func TestNewSimpleFormFSM(t *testing.T) {
	fsm := NewSimpleFormFSM()

	tests := []struct {
		name          string
		currentState  string
		action        string
		wantNextState string
		wantTaskState State
		wantErr       bool
	}{
		// START
		{
			name:          "start from empty — no task state change",
			currentState:  "",
			action:        FSMActionStart,
			wantNextState: string(Initialized),
			wantTaskState: "",
		},
		// DRAFT
		{
			name:          "draft from initialised",
			currentState:  string(Initialized),
			action:        SimpleFormActionDraft,
			wantNextState: string(TraderSavedAsDraft),
			wantTaskState: InProgress,
		},
		{
			name:          "draft from draft",
			currentState:  string(TraderSavedAsDraft),
			action:        SimpleFormActionDraft,
			wantNextState: string(TraderSavedAsDraft),
			wantTaskState: InProgress,
		},
		// SUBMIT (no OGA)
		{
			name:          "submit complete from initialised",
			currentState:  string(Initialized),
			action:        simpleFormFSMSubmitComplete,
			wantNextState: string(TraderSubmitted),
			wantTaskState: Completed,
		},
		{
			name:          "submit complete from draft",
			currentState:  string(TraderSavedAsDraft),
			action:        simpleFormFSMSubmitComplete,
			wantNextState: string(TraderSubmitted),
			wantTaskState: Completed,
		},
		// SUBMIT (await OGA)
		{
			name:          "submit await oga from initialised",
			currentState:  string(Initialized),
			action:        simpleFormFSMSubmitAwaitOGA,
			wantNextState: string(OGAAcknowledged),
			wantTaskState: InProgress,
		},
		{
			name:          "submit await oga from draft",
			currentState:  string(TraderSavedAsDraft),
			action:        simpleFormFSMSubmitAwaitOGA,
			wantNextState: string(OGAAcknowledged),
			wantTaskState: InProgress,
		},
		// OGA outcomes
		{
			name:          "oga approved",
			currentState:  string(OGAAcknowledged),
			action:        simpleFormFSMOgaApproved,
			wantNextState: string(OGAReviewed),
			wantTaskState: Completed,
		},
		{
			name:          "oga rejected",
			currentState:  string(OGAAcknowledged),
			action:        simpleFormFSMOgaRejected,
			wantNextState: string(OGAReviewed),
			wantTaskState: Failed,
		},
		// Invalid transitions
		{
			name:         "draft not permitted from submitted",
			currentState: string(TraderSubmitted),
			action:       SimpleFormActionDraft,
			wantErr:      true,
		},
		{
			name:         "oga approved not permitted before oga acknowledged",
			currentState: string(Initialized),
			action:       simpleFormFSMOgaApproved,
			wantErr:      true,
		},
		{
			name:         "start not permitted twice",
			currentState: string(Initialized),
			action:       FSMActionStart,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := fsm.Transition(tc.currentState, tc.action)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if outcome.NextPluginState != tc.wantNextState {
				t.Errorf("NextPluginState: got %q, want %q", outcome.NextPluginState, tc.wantNextState)
			}
			if outcome.NextTaskState != tc.wantTaskState {
				t.Errorf("NextTaskState: got %q, want %q", outcome.NextTaskState, tc.wantTaskState)
			}
		})
	}
}

func TestNewWaitForEventFSM(t *testing.T) {
	fsm := NewWaitForEventFSM()

	tests := []struct {
		name          string
		currentState  string
		action        string
		wantNextState string
		wantTaskState State
		wantErr       bool
	}{
		{
			name:          "start moves to in-progress",
			currentState:  "",
			action:        FSMActionStart,
			wantNextState: string(notifiedService),
			wantTaskState: InProgress,
		},
		{
			name:          "complete from notified service",
			currentState:  string(notifiedService),
			action:        "complete",
			wantNextState: string(receivedCallback),
			wantTaskState: Completed,
		},
		{
			name:         "complete not permitted before start",
			currentState: "",
			action:       "complete",
			wantErr:      true,
		},
		{
			name:         "start not permitted after notified",
			currentState: string(notifiedService),
			action:       FSMActionStart,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outcome, err := fsm.Transition(tc.currentState, tc.action)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if outcome.NextPluginState != tc.wantNextState {
				t.Errorf("NextPluginState: got %q, want %q", outcome.NextPluginState, tc.wantNextState)
			}
			if outcome.NextTaskState != tc.wantTaskState {
				t.Errorf("NextTaskState: got %q, want %q", outcome.NextTaskState, tc.wantTaskState)
			}
		})
	}
}

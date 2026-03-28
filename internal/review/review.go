// Package review provides PASS/FAIL signal parsing and consensus logic.
package review

import (
	"regexp"
	"strings"

	"github.com/plexusone/agentpair/internal/bridge"
)

// Signal represents a review signal.
type Signal string

const (
	SignalPass    Signal = "PASS"
	SignalFail    Signal = "FAIL"
	SignalNone    Signal = ""
	SignalPending Signal = "PENDING"
)

// Result represents the result of signal parsing.
type Result struct {
	Signal   Signal
	Reason   string
	Raw      string
	Agent    string
	Explicit bool // Whether the signal was explicitly stated
}

// Parser parses review signals from agent output.
type Parser struct {
	doneSignal string
	patterns   []*regexp.Regexp
}

// NewParser creates a new signal parser.
func NewParser(doneSignal string) *Parser {
	if doneSignal == "" {
		doneSignal = "DONE"
	}

	patterns := []*regexp.Regexp{
		// Explicit PASS/FAIL patterns
		regexp.MustCompile(`(?i)\b(PASS|FAIL)\b`),
		// Review outcome patterns
		regexp.MustCompile(`(?i)review\s+(passed|failed|pass|fail)`),
		regexp.MustCompile(`(?i)(approved|rejected|approve|reject)`),
		// Done signal pattern
		regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(doneSignal) + `\b`),
	}

	return &Parser{
		doneSignal: doneSignal,
		patterns:   patterns,
	}
}

// Parse extracts a signal from the given text.
func (p *Parser) Parse(text string) *Result {
	result := &Result{
		Signal: SignalNone,
		Raw:    text,
	}

	// Check for explicit PASS
	if p.containsPass(text) {
		result.Signal = SignalPass
		result.Explicit = true
		result.Reason = p.extractReason(text, "PASS")
		return result
	}

	// Check for explicit FAIL
	if p.containsFail(text) {
		result.Signal = SignalFail
		result.Explicit = true
		result.Reason = p.extractReason(text, "FAIL")
		return result
	}

	// Check for approval patterns
	if p.containsApproval(text) {
		result.Signal = SignalPass
		result.Reason = "implicit approval detected"
		return result
	}

	// Check for rejection patterns
	if p.containsRejection(text) {
		result.Signal = SignalFail
		result.Reason = "implicit rejection detected"
		return result
	}

	return result
}

// ParseMessage parses a signal from a bridge message.
func (p *Parser) ParseMessage(msg *bridge.Message) *Result {
	result := p.Parse(msg.Content)
	result.Agent = msg.From

	// Check for explicit signal messages
	if msg.Type == bridge.TypeSignal {
		switch msg.Signal {
		case bridge.SignalPass:
			result.Signal = SignalPass
			result.Explicit = true
		case bridge.SignalFail:
			result.Signal = SignalFail
			result.Explicit = true
		case bridge.SignalDone:
			result.Signal = SignalPass // DONE implies task complete
			result.Explicit = true
		}
	}

	return result
}

func (p *Parser) containsPass(text string) bool {
	upper := strings.ToUpper(text)
	return strings.Contains(upper, "PASS") ||
		strings.Contains(upper, "APPROVED") ||
		strings.Contains(upper, "LGTM")
}

func (p *Parser) containsFail(text string) bool {
	upper := strings.ToUpper(text)
	return strings.Contains(upper, "FAIL") ||
		strings.Contains(upper, "REJECTED") ||
		strings.Contains(upper, "NEEDS CHANGES")
}

func (p *Parser) containsApproval(text string) bool {
	lower := strings.ToLower(text)
	approvalPhrases := []string{
		"looks good",
		"well done",
		"correctly implemented",
		"works as expected",
		"tests pass",
		"all tests pass",
	}
	for _, phrase := range approvalPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func (p *Parser) containsRejection(text string) bool {
	lower := strings.ToLower(text)
	rejectionPhrases := []string{
		"needs work",
		"needs changes",
		"does not work",
		"doesn't work",
		"tests fail",
		"test failed",
		"incorrect",
		"bug found",
		"issue found",
	}
	for _, phrase := range rejectionPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func (p *Parser) extractReason(text, signal string) string {
	// Try to extract the reason after the signal
	upper := strings.ToUpper(text)
	idx := strings.Index(upper, signal)
	if idx == -1 {
		return ""
	}

	// Get text after the signal
	remaining := text[idx+len(signal):]

	// Clean up common separators
	remaining = strings.TrimLeft(remaining, ":- \t\n")

	// Take first line or sentence
	if idx := strings.IndexAny(remaining, "\n."); idx != -1 {
		remaining = remaining[:idx]
	}

	// Limit length
	if len(remaining) > 200 {
		remaining = remaining[:200] + "..."
	}

	return strings.TrimSpace(remaining)
}

// Consensus determines the overall review result from multiple agents.
type Consensus struct {
	mode string // "claude", "codex", or "claudex"
}

// NewConsensus creates a new consensus calculator.
func NewConsensus(mode string) *Consensus {
	return &Consensus{mode: mode}
}

// ConsensusResult represents the consensus outcome.
type ConsensusResult struct {
	Signal       Signal
	Reason       string
	ClaudeSignal Signal
	CodexSignal  Signal
	Unanimous    bool
}

// Calculate determines consensus from agent results.
func (c *Consensus) Calculate(claudeResult, codexResult *Result) *ConsensusResult {
	result := &ConsensusResult{
		ClaudeSignal: SignalNone,
		CodexSignal:  SignalNone,
	}

	if claudeResult != nil {
		result.ClaudeSignal = claudeResult.Signal
	}
	if codexResult != nil {
		result.CodexSignal = codexResult.Signal
	}

	switch c.mode {
	case "claude":
		// Claude's decision is final
		result.Signal = result.ClaudeSignal
		if claudeResult != nil {
			result.Reason = claudeResult.Reason
		}
		result.Unanimous = true

	case "codex":
		// Codex's decision is final
		result.Signal = result.CodexSignal
		if codexResult != nil {
			result.Reason = codexResult.Reason
		}
		result.Unanimous = true

	case "claudex":
		// Both must agree for PASS, any FAIL means FAIL
		if result.ClaudeSignal == SignalFail || result.CodexSignal == SignalFail {
			result.Signal = SignalFail
			result.Unanimous = result.ClaudeSignal == result.CodexSignal
			if result.ClaudeSignal == SignalFail && claudeResult != nil {
				result.Reason = "Claude: " + claudeResult.Reason
			}
			if result.CodexSignal == SignalFail && codexResult != nil {
				if result.Reason != "" {
					result.Reason += "; "
				}
				result.Reason += "Codex: " + codexResult.Reason
			}
		} else if result.ClaudeSignal == SignalPass && result.CodexSignal == SignalPass {
			result.Signal = SignalPass
			result.Unanimous = true
			result.Reason = "Both agents approved"
		} else {
			// Mixed or incomplete results
			result.Signal = SignalPending
			result.Reason = "Waiting for both agents to provide review"
		}
	}

	return result
}

// IsDone checks if the done signal is present in text.
func (p *Parser) IsDone(text string) bool {
	upper := strings.ToUpper(text)
	doneUpper := strings.ToUpper(p.doneSignal)
	return strings.Contains(upper, doneUpper)
}

// ContainsDoneSignal checks messages for the done signal.
func ContainsDoneSignal(msgs []*bridge.Message, doneSignal string) bool {
	parser := NewParser(doneSignal)
	for _, msg := range msgs {
		if msg.IsDoneSignal() || parser.IsDone(msg.Content) {
			return true
		}
	}
	return false
}

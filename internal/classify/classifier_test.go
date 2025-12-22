package classify

import (
	"testing"

	"github.com/solvaholic/threadmine/internal/normalize"
)

func TestClassifyQuestion(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectQuestion bool
		minConfidence  float64
	}{
		{
			name:           "simple question with question mark",
			content:        "How do I configure this?",
			expectQuestion: true,
			minConfidence:  0.8,
		},
		{
			name:           "question without question mark",
			content:        "anyone know how to fix this",
			expectQuestion: true,
			minConfidence:  0.3,
		},
		{
			name:           "help seeking",
			content:        "I'm stuck trying to get this working",
			expectQuestion: true,
			minConfidence:  0.2,
		},
		{
			name:           "not a question - statement about solution",
			content:        "Here's the solution for that.",
			expectQuestion: false,
		},
		{
			name:           "statement",
			content:        "The server is running fine now.",
			expectQuestion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &normalize.NormalizedMessage{
				Content: tt.content,
			}

			result := classifyQuestion(msg)

			if tt.expectQuestion && result == nil {
				t.Errorf("expected question classification, got nil")
				return
			}

			if !tt.expectQuestion && result != nil {
				t.Errorf("expected no classification, got %v", result)
				return
			}

			if result != nil {
				if result.Type != "question" {
					t.Errorf("expected type 'question', got '%s'", result.Type)
				}
				if result.Confidence < tt.minConfidence {
					t.Errorf("expected confidence >= %.2f, got %.2f", tt.minConfidence, result.Confidence)
				}
				if len(result.Signals) == 0 {
					t.Errorf("expected at least one signal")
				}
			}
		})
	}
}

func TestClassifySolution(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		codeBlocks     []normalize.CodeBlock
		urls           []string
		expectSolution bool
		minConfidence  float64
	}{
		{
			name:    "code block with instruction",
			content: "Try this fix:",
			codeBlocks: []normalize.CodeBlock{
				{Language: "go", Code: "fmt.Println(\"hello\")"},
			},
			expectSolution: true,
			minConfidence:  0.6,
		},
		{
			name:           "step-by-step instructions",
			content:        "Here's how to fix it:\n1. First do this\n2. Then do that\n3. Finally restart",
			expectSolution: true,
			minConfidence:  0.4,
		},
		{
			name:           "documentation reference",
			content:        "Check out the docs here",
			urls:           []string{"https://docs.example.com/guide"},
			expectSolution: true,
			minConfidence:  0.25,
		},
		{
			name:           "not a solution",
			content:        "I have the same problem too",
			expectSolution: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &normalize.NormalizedMessage{
				Content:    tt.content,
				CodeBlocks: tt.codeBlocks,
				URLs:       tt.urls,
			}

			result := classifySolution(msg)

			if tt.expectSolution && result == nil {
				t.Errorf("expected solution classification, got nil")
				return
			}

			if !tt.expectSolution && result != nil {
				t.Errorf("expected no classification, got %v", result)
				return
			}

			if result != nil {
				if result.Type != "solution" {
					t.Errorf("expected type 'solution', got '%s'", result.Type)
				}
				if result.Confidence < tt.minConfidence {
					t.Errorf("expected confidence >= %.2f, got %.2f", tt.minConfidence, result.Confidence)
				}
			}
		})
	}
}

func TestClassifyAcknowledgment(t *testing.T) {
	tests := []struct {
		name                  string
		content               string
		expectAcknowledgment  bool
		minConfidence         float64
	}{
		{
			name:                 "thanks and success",
			content:              "Thanks! That worked perfectly.",
			expectAcknowledgment: true,
			minConfidence:        0.6,
		},
		{
			name:                 "simple thumbs up",
			content:              "👍",
			expectAcknowledgment: true,
			minConfidence:        0.2,
		},
		{
			name:                 "confirmation",
			content:              "That fixed it, thanks!",
			expectAcknowledgment: true,
			minConfidence:        0.6,
		},
		{
			name:                 "not acknowledgment",
			content:              "I still have the same issue",
			expectAcknowledgment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &normalize.NormalizedMessage{
				Content: tt.content,
			}

			result := classifyAcknowledgment(msg)

			if tt.expectAcknowledgment && result == nil {
				t.Errorf("expected acknowledgment classification, got nil")
				return
			}

			if !tt.expectAcknowledgment && result != nil {
				t.Errorf("expected no classification, got %v", result)
				return
			}

			if result != nil {
				if result.Type != "acknowledgment" {
					t.Errorf("expected type 'acknowledgment', got '%s'", result.Type)
				}
				if result.Confidence < tt.minConfidence {
					t.Errorf("expected confidence >= %.2f, got %.2f", tt.minConfidence, result.Confidence)
				}
			}
		})
	}
}

func TestClassifyAnswer(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		context       *ThreadContext
		expectAnswer  bool
		minConfidence float64
	}{
		{
			name:    "answer in question thread",
			content: "You can fix this by updating your config file",
			context: &ThreadContext{
				HasQuestion:  true,
				IsThreadRoot: false,
				Position:     1,
			},
			expectAnswer:  true,
			minConfidence: 0.4,
		},
		{
			name:    "thread root cannot be answer",
			content: "You should try this approach",
			context: &ThreadContext{
				HasQuestion:  true,
				IsThreadRoot: true,
				Position:     0,
			},
			expectAnswer: false,
		},
		{
			name:    "not in question thread",
			content: "You can do it this way",
			context: &ThreadContext{
				HasQuestion:  false,
				IsThreadRoot: false,
				Position:     1,
			},
			expectAnswer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &normalize.NormalizedMessage{
				Content: tt.content,
			}

			result := classifyAnswer(msg, tt.context)

			if tt.expectAnswer && result == nil {
				t.Errorf("expected answer classification, got nil")
				return
			}

			if !tt.expectAnswer && result != nil {
				t.Errorf("expected no classification, got %v", result)
				return
			}

			if result != nil {
				if result.Type != "answer" {
					t.Errorf("expected type 'answer', got '%s'", result.Type)
				}
				if result.Confidence < tt.minConfidence {
					t.Errorf("expected confidence >= %.2f, got %.2f", tt.minConfidence, result.Confidence)
				}
			}
		})
	}
}

func TestClassifyMessage_Multiple(t *testing.T) {
	// A message can have multiple classifications
	msg := &normalize.NormalizedMessage{
		Content: "Try this solution:",
		CodeBlocks: []normalize.CodeBlock{
			{Language: "bash", Code: "npm install"},
		},
	}

	ctx := &ThreadContext{
		HasQuestion:  true,
		IsThreadRoot: false,
		Position:     1,
	}

	classifications := ClassifyMessage(msg, ctx)

	// Should be classified as both answer and solution
	hasAnswer := false
	hasSolution := false

	for _, c := range classifications {
		if c.Type == "answer" {
			hasAnswer = true
		}
		if c.Type == "solution" {
			hasSolution = true
		}
	}

	if !hasAnswer {
		t.Errorf("expected answer classification")
	}
	if !hasSolution {
		t.Errorf("expected solution classification")
	}
}

package classify

import (
	"regexp"
	"strings"

	"github.com/solvaholic/threadmine/internal/normalize"
)

// Classification represents a message classification with confidence score
type Classification struct {
	Type       string   `json:"type"`        // "question", "answer", "solution", "acknowledgment"
	Confidence float64  `json:"confidence"`  // 0.0 to 1.0
	Signals    []string `json:"signals"`     // What triggered this classification
}

// ClassifyMessage analyzes a message and returns all applicable classifications
func ClassifyMessage(msg *normalize.NormalizedMessage, threadContext *ThreadContext) []Classification {
	var classifications []Classification

	// Check for question
	if q := classifyQuestion(msg); q != nil {
		classifications = append(classifications, *q)
	}

	// Check for answer (requires thread context)
	if threadContext != nil {
		if a := classifyAnswer(msg, threadContext); a != nil {
			classifications = append(classifications, *a)
		}
	}

	// Check for solution
	if s := classifySolution(msg); s != nil {
		classifications = append(classifications, *s)
	}

	// Check for acknowledgment
	if ack := classifyAcknowledgment(msg); ack != nil {
		classifications = append(classifications, *ack)
	}

	return classifications
}

// ThreadContext provides context about the thread for better classification
type ThreadContext struct {
	HasQuestion    bool   // Does the thread contain a question?
	QuestionAuthor string // Who asked the question?
	IsThreadRoot   bool   // Is this the first message in the thread?
	Position       int    // Position in thread (0 = root)
}

// classifyQuestion detects if a message is asking a question
func classifyQuestion(msg *normalize.NormalizedMessage) *Classification {
	content := strings.ToLower(msg.Content)
	var signals []string
	confidence := 0.0

	// Strong signal: Contains question mark
	if strings.Contains(content, "?") {
		signals = append(signals, "question_mark")
		confidence += 0.4
	}

	// Question words at start
	questionStarters := []string{
		"how do i", "how can i", "how to", "how would",
		"what is", "what's", "what are", "what if",
		"where is", "where can", "where do",
		"when should", "when do", "when is",
		"why does", "why is", "why would",
		"who can", "who is", "who knows",
		"can someone", "can anyone", "could someone",
		"is there", "are there",
		"does anyone", "does someone",
		"has anyone", "has someone",
		"should i", "would it",
		"any ideas", "anyone know",
	}

	for _, starter := range questionStarters {
		if strings.HasPrefix(content, starter) {
			signals = append(signals, "question_starter:"+starter)
			confidence += 0.5
			break
		}
	}

	// Help-seeking phrases
	helpPhrases := []string{
		"help", "stuck", "issue", "problem", "error",
		"not working", "doesn't work", "can't get", "unable to",
		"trying to", "need to", "want to",
		"anyone else", "has anyone",
	}

	for _, phrase := range helpPhrases {
		if strings.Contains(content, phrase) {
			signals = append(signals, "help_phrase:"+phrase)
			confidence += 0.2
			break
		}
	}

	// Moderate confidence if we have signals
	if len(signals) == 0 {
		return nil
	}

	// Cap confidence at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Minimum confidence threshold
	if confidence < 0.2 {
		return nil
	}

	return &Classification{
		Type:       "question",
		Confidence: confidence,
		Signals:    signals,
	}
}

// classifyAnswer detects if a message is providing an answer
func classifyAnswer(msg *normalize.NormalizedMessage, ctx *ThreadContext) *Classification {
	// Must be in a thread with a question
	if !ctx.HasQuestion {
		return nil
	}

	// Thread root can't be an answer (it's usually the question)
	if ctx.IsThreadRoot {
		return nil
	}

	// Don't classify the question author's own messages as answers (unless they answer themselves)
	// For now, assume others' messages in a question thread are potential answers
	
	content := strings.ToLower(msg.Content)
	var signals []string
	confidence := 0.0

	// Base confidence for being in a thread with a question
	confidence = 0.3
	signals = append(signals, "in_question_thread")

	// Answer phrases
	answerPhrases := []string{
		"you can", "you should", "you need to", "you have to",
		"try this", "try using", "try adding",
		"the solution", "the answer", "the fix",
		"i think", "i believe", "i suggest",
		"here's how", "here is how",
		"check out", "take a look",
		"you might want", "you could",
	}

	for _, phrase := range answerPhrases {
		if strings.Contains(content, phrase) {
			signals = append(signals, "answer_phrase:"+phrase)
			confidence += 0.2
			break
		}
	}

	// Contains code (likely a solution)
	if len(msg.CodeBlocks) > 0 {
		signals = append(signals, "contains_code")
		confidence += 0.2
	}

	// Contains URLs (documentation, examples)
	if len(msg.URLs) > 0 {
		signals = append(signals, "contains_urls")
		confidence += 0.1
	}

	// Longer messages are more likely to be substantive answers
	if len(msg.Content) > 100 {
		signals = append(signals, "substantive_length")
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Must have some answer signals beyond just being in thread
	if len(signals) <= 1 {
		return nil
	}

	return &Classification{
		Type:       "answer",
		Confidence: confidence,
		Signals:    signals,
	}
}

// classifySolution detects if a message contains a proposed solution
func classifySolution(msg *normalize.NormalizedMessage) *Classification {
	content := strings.ToLower(msg.Content)
	var signals []string
	confidence := 0.0

	// Strong signal: Contains code blocks
	if len(msg.CodeBlocks) > 0 {
		signals = append(signals, "code_block")
		confidence += 0.4
	}

	// Solution indicator phrases
	solutionPhrases := []string{
		"try this", "try adding", "try changing",
		"here's a fix", "here's the fix", "here's how",
		"you can fix", "to fix this", "the fix is",
		"the solution", "solution is",
		"this should work", "this will work",
		"this fixes", "this solved",
	}

	for _, phrase := range solutionPhrases {
		if strings.Contains(content, phrase) {
			signals = append(signals, "solution_phrase:"+phrase)
			confidence += 0.3
			break
		}
	}

	// Instructions/steps (numbered or bullet points)
	stepPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^1\.|^1\)`),           // Numbered list starting with 1
		regexp.MustCompile(`(?m)^- |^\* |^• `),        // Bullet points
		regexp.MustCompile(`first.*then.*finally`),    // Sequential instructions
		regexp.MustCompile(`step \d+`),                // "step 1", "step 2"
	}

	for _, pattern := range stepPatterns {
		if pattern.MatchString(content) {
			signals = append(signals, "contains_steps")
			confidence += 0.2
			break
		}
	}

	// Documentation/reference URLs
	docPatterns := []string{
		"docs.", "documentation", "/docs/", 
		"stackoverflow", "github.com",
		"tutorial", "example", "guide",
	}

	for _, url := range msg.URLs {
		urlLower := strings.ToLower(url)
		for _, pattern := range docPatterns {
			if strings.Contains(urlLower, pattern) {
				signals = append(signals, "reference_url")
				confidence += 0.25
				break
			}
		}
	}

	if len(signals) == 0 {
		return nil
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Minimum threshold
	if confidence < 0.25 {
		return nil
	}

	return &Classification{
		Type:       "solution",
		Confidence: confidence,
		Signals:    signals,
	}
}

// classifyAcknowledgment detects if a message is acknowledging/accepting a solution
func classifyAcknowledgment(msg *normalize.NormalizedMessage) *Classification {
	content := strings.ToLower(msg.Content)
	var signals []string
	confidence := 0.0

	// Thank you phrases
	thankPhrases := []string{
		"thank", "thanks", "thx", "ty",
		"appreciate", "grateful",
	}

	for _, phrase := range thankPhrases {
		if strings.Contains(content, phrase) {
			signals = append(signals, "thanks:"+phrase)
			confidence += 0.3
			break
		}
	}

	// Success indicators
	successPhrases := []string{
		"worked", "works", "working",
		"fixed", "solved", "resolved",
		"that did it", "that worked", "that fixed",
		"perfect", "exactly what i needed",
		"this solved", "this fixed",
		"got it working", "got it to work",
	}

	for _, phrase := range successPhrases {
		if strings.Contains(content, phrase) {
			signals = append(signals, "success:"+phrase)
			confidence += 0.4
			break
		}
	}

	// Positive reactions (emoji/symbols)
	positiveIndicators := []string{
		"👍", "✓", "✔", ":+1:", ":check:", ":white_check_mark:",
		"awesome", "great", "excellent", "brilliant",
	}

	for _, indicator := range positiveIndicators {
		if strings.Contains(content, indicator) || strings.Contains(msg.Content, indicator) {
			signals = append(signals, "positive_indicator")
			confidence += 0.2
			break
		}
	}

	// Short affirmative messages
	affirmatives := []string{
		"yes!", "yep", "yup", "yeah",
		"perfect", "exactly",
	}

	contentTrimmed := strings.TrimSpace(content)
	for _, affirm := range affirmatives {
		if contentTrimmed == affirm || contentTrimmed == affirm+"!" {
			signals = append(signals, "affirmative:"+affirm)
			confidence += 0.25
			break
		}
	}

	if len(signals) == 0 {
		return nil
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Minimum threshold (lower to catch simple emoji reactions)
	if confidence < 0.2 {
		return nil
	}

	return &Classification{
		Type:       "acknowledgment",
		Confidence: confidence,
		Signals:    signals,
	}
}

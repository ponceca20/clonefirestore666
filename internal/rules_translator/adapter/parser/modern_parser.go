package parser

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"firestore-clone/internal/rules_translator/domain"
)

// TokenType representa los tipos de tokens
type TokenType int

const (
	// Literals
	IDENTIFIER TokenType = iota
	STRING
	NUMBER

	// Keywords
	RULES_VERSION
	SERVICE
	MATCH
	ALLOW
	DENY
	IF

	// Operators
	EQUALS
	NOT_EQUALS
	AND
	OR
	NOT
	LESS_THAN
	GREATER_THAN
	LESS_EQUAL
	GREATER_EQUAL
	DOT
	PLUS
	MINUS
	DOLLAR
	SEMICOLON
	COLON
	COMMA

	// Delimiters
	LBRACE
	RBRACE
	LPAREN
	RPAREN
	LBRACKET
	RBRACKET

	// Special
	PATH
	EOF
	INVALID
)

// Token representa un token en el código
type Token struct {
	Type     TokenType
	Value    string
	Line     int
	Column   int
	Position int
}

// Position representa una posición en el código fuente
type Position struct {
	Line   int
	Column int
	Offset int
}

// Lexer convierte texto en tokens
type Lexer struct {
	input    string
	position int
	line     int
	column   int
	tokens   []Token
}

// NewLexer crea un nuevo lexer
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		line:   1,
		column: 1,
		tokens: make([]Token, 0),
	}
}

// Tokenize convierte el input en tokens
func (l *Lexer) Tokenize() ([]Token, error) {
	for l.position < len(l.input) {
		// Skip whitespace
		if l.isWhitespace(l.current()) {
			if l.current() == '\n' {
				l.line++
				l.column = 1
			} else {
				l.column++
			}
			l.advance()
			continue
		}
		// Skip comments
		if l.current() == '/' && l.peek() == '/' {
			l.skipLineComment()
			continue
		}

		// Skip multi-line comments
		if l.current() == '/' && l.peek() == '*' {
			l.skipMultiLineComment()
			continue
		}

		// Tokenize
		token, err := l.nextToken()
		if err != nil {
			return nil, err
		}

		if token.Type != EOF {
			l.tokens = append(l.tokens, token)
		}
	}

	// Add EOF token
	l.tokens = append(l.tokens, Token{
		Type:     EOF,
		Value:    "",
		Line:     l.line,
		Column:   l.column,
		Position: l.position,
	})

	return l.tokens, nil
}

// nextToken extrae el siguiente token
func (l *Lexer) nextToken() (Token, error) {
	startLine := l.line
	startColumn := l.column

	ch := l.current()
	switch ch {
	case '{':
		l.advance()
		return l.makeToken(LBRACE, "{", startLine, startColumn), nil
	case '}':
		l.advance()
		return l.makeToken(RBRACE, "}", startLine, startColumn), nil
	case '(':
		l.advance()
		return l.makeToken(LPAREN, "(", startLine, startColumn), nil
	case ')':
		l.advance()
		return l.makeToken(RPAREN, ")", startLine, startColumn), nil
	case '=':
		l.advance()
		if l.current() == '=' {
			l.advance()
			return l.makeToken(EQUALS, "==", startLine, startColumn), nil
		}
		return l.makeToken(EQUALS, "=", startLine, startColumn), nil
	case '!':
		l.advance()
		if l.current() == '=' {
			l.advance()
			return l.makeToken(NOT_EQUALS, "!=", startLine, startColumn), nil
		}
		return l.makeToken(NOT, "!", startLine, startColumn), nil
	case '&':
		l.advance()
		if l.current() == '&' {
			l.advance()
			return l.makeToken(AND, "&&", startLine, startColumn), nil
		}
		return l.makeToken(AND, "&", startLine, startColumn), nil
	case '|':
		l.advance()
		if l.current() == '|' {
			l.advance()
			return l.makeToken(OR, "||", startLine, startColumn), nil
		}
		return l.makeToken(OR, "|", startLine, startColumn), nil
	case '<':
		l.advance()
		if l.current() == '=' {
			l.advance()
			return l.makeToken(LESS_EQUAL, "<=", startLine, startColumn), nil
		}
		return l.makeToken(LESS_THAN, "<", startLine, startColumn), nil
	case '>':
		l.advance()
		if l.current() == '=' {
			l.advance()
			return l.makeToken(GREATER_EQUAL, ">=", startLine, startColumn), nil
		}
		return l.makeToken(GREATER_THAN, ">", startLine, startColumn), nil
	case ';':
		l.advance()
		return l.makeToken(SEMICOLON, ";", startLine, startColumn), nil
	case ':':
		l.advance()
		return l.makeToken(COLON, ":", startLine, startColumn), nil
	case ',':
		l.advance()
		return l.makeToken(COMMA, ",", startLine, startColumn), nil
	case '.':
		l.advance()
		return l.makeToken(DOT, ".", startLine, startColumn), nil
	case '+':
		l.advance()
		return l.makeToken(PLUS, "+", startLine, startColumn), nil
	case '-':
		l.advance()
		return l.makeToken(MINUS, "-", startLine, startColumn), nil
	case '[':
		l.advance()
		return l.makeToken(LBRACKET, "[", startLine, startColumn), nil
	case ']':
		l.advance()
		return l.makeToken(RBRACKET, "]", startLine, startColumn), nil
	case '$':
		l.advance()
		return l.makeToken(DOLLAR, "$", startLine, startColumn), nil
	case '"', '\'':
		return l.readString()
	case '/':
		// Path segment
		return l.readPath()
	case 0:
		return l.makeToken(EOF, "", startLine, startColumn), nil
	default:
		if l.isLetter(ch) {
			return l.readIdentifier()
		} else if l.isDigit(ch) {
			return l.readNumber()
		}

		return Token{}, fmt.Errorf("unexpected character '%c' at line %d, column %d",
			ch, l.line, l.column)
	}
}

// ModernParser usa técnicas modernas de parsing
type ModernParser struct {
	tokens  []Token
	current int
	errors  []domain.ParseError

	// Metrics fields
	totalParsed    int64
	lastParseTime  time.Time
	totalParseTime time.Duration
	parseErrors    int64
}

// NewModernParser crea un nuevo parser moderno
func NewModernParser() *ModernParser {
	return &ModernParser{
		errors: make([]domain.ParseError, 0),
	}
}

// NewModernParser crea una nueva instancia del parser moderno
func NewModernParserInstance() domain.RulesParser {
	return NewModernParser()
}

// Parse parsea el contenido usando técnicas modernas
func (p *ModernParser) Parse(ctx context.Context, content io.Reader) (*domain.ParseResult, error) {
	startTime := time.Now()

	// Leer contenido
	var sb strings.Builder
	if _, err := io.Copy(&sb, content); err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	result, err := p.ParseString(ctx, sb.String())
	if err != nil {
		return nil, err
	}

	result.ParseTime = time.Since(startTime)
	return result, nil
}

// ParseString parsea el contenido desde string usando técnicas modernas
func (p *ModernParser) ParseString(ctx context.Context, content string) (*domain.ParseResult, error) {
	// Update metrics at the start
	startTime := time.Now()
	p.totalParsed++

	// Fase 1: Lexical Analysis
	lexer := NewLexer(content)
	tokens, err := lexer.Tokenize()
	if err != nil {
		p.parseErrors++
		return nil, fmt.Errorf("lexical error: %w", err)
	}

	p.tokens = tokens
	p.current = 0

	// Fase 2: Syntactic Analysis
	ruleset, err := p.parseRuleset()
	if err != nil {
		p.parseErrors++
		return nil, fmt.Errorf("syntax error: %w", err)
	}

	// Fase 3: Semantic Analysis
	if err := p.validateSemantics(ruleset); err != nil {
		p.parseErrors++
		return nil, fmt.Errorf("semantic error: %w", err)
	}

	// Update metrics at the end
	duration := time.Since(startTime)
	p.lastParseTime = time.Now()
	p.totalParseTime += duration

	result := &domain.ParseResult{
		Ruleset:   ruleset,
		Errors:    p.errors,
		RuleCount: p.countRules(ruleset),
		LineCount: p.countLines(content),
	}

	return result, nil
}

// parseRuleset parsea el ruleset completo
func (p *ModernParser) parseRuleset() (*domain.FirestoreRuleset, error) {
	ruleset := &domain.FirestoreRuleset{
		Version: "2", // default
		Matches: make([]*domain.MatchBlock, 0),
	}

	// rules_version (opcional)
	if p.check(RULES_VERSION) {
		version, err := p.parseRulesVersion()
		if err != nil {
			return nil, err
		}
		ruleset.Version = version
	}

	// service
	if !p.check(SERVICE) {
		return nil, p.error("expected 'service' declaration")
	}

	service, matches, err := p.parseService()
	if err != nil {
		return nil, err
	}

	ruleset.Service = service
	ruleset.Matches = matches

	return ruleset, nil
}

// parseService parsea la declaración service
func (p *ModernParser) parseService() (string, []*domain.MatchBlock, error) {
	// service
	if !p.consume(SERVICE) {
		return "", nil, p.error("expected 'service'")
	}

	// service name
	if !p.check(IDENTIFIER) {
		return "", nil, p.error("expected service name")
	}

	serviceName := p.advance().Value

	// {
	if !p.consume(LBRACE) {
		return "", nil, p.error("expected '{'")
	}

	// match blocks
	matches := make([]*domain.MatchBlock, 0)

	for !p.check(RBRACE) && !p.isAtEnd() {
		match, err := p.parseMatchBlock()
		if err != nil {
			return "", nil, err
		}
		matches = append(matches, match)
	}
	// }
	if !p.consume(RBRACE) {
		return "", nil, p.error("unclosed service block - expected '}'")
	}

	return serviceName, matches, nil
}

// Helper methods
func (l *Lexer) current() byte {
	if l.position >= len(l.input) {
		return 0
	}
	return l.input[l.position]
}

func (l *Lexer) peek() byte {
	if l.position+1 >= len(l.input) {
		return 0
	}
	return l.input[l.position+1]
}

func (l *Lexer) advance() {
	if l.position < len(l.input) {
		l.position++
		l.column++
	}
}

func (l *Lexer) isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func (l *Lexer) isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func (l *Lexer) isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func (l *Lexer) isAlphaNumeric(ch byte) bool {
	return l.isLetter(ch) || l.isDigit(ch)
}

func (l *Lexer) makeToken(tokenType TokenType, value string, line, column int) Token {
	return Token{
		Type:     tokenType,
		Value:    value,
		Line:     line,
		Column:   column,
		Position: l.position,
	}
}

func (l *Lexer) skipLineComment() {
	for l.current() != '\n' && l.current() != 0 {
		l.advance()
	}
}

func (l *Lexer) skipMultiLineComment() {
	l.advance() // Skip '/'
	l.advance() // Skip '*'

	for l.current() != 0 {
		if l.current() == '*' && l.peek() == '/' {
			l.advance() // Skip '*'
			l.advance() // Skip '/'
			break
		}

		if l.current() == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
		l.advance()
	}
}

func (l *Lexer) readString() (Token, error) {
	quote := l.current()
	startLine := l.line
	startColumn := l.column
	l.advance() // Skip opening quote

	value := ""
	for l.current() != quote && l.current() != 0 {
		if l.current() == '\n' {
			return Token{}, fmt.Errorf("unterminated string at line %d", l.line)
		}
		value += string(l.current())
		l.advance()
	}

	if l.current() == 0 {
		return Token{}, fmt.Errorf("unterminated string at line %d", l.line)
	}

	l.advance() // Skip closing quote
	return l.makeToken(STRING, value, startLine, startColumn), nil
}

func (l *Lexer) readPath() (Token, error) {
	startLine := l.line
	startColumn := l.column
	value := "" // Read complete path including variables like /databases/{database}/documents/{document=**}
	// Stop at semicolon, closing parenthesis, or whitespace
	// Also handle $(variable) interpolation within paths
	for l.current() != 0 && !l.isWhitespace(l.current()) && l.current() != ';' && l.current() != ')' {
		if l.current() == '{' {
			// Read variable completely
			for l.current() != 0 && l.current() != '}' {
				value += string(l.current())
				l.advance()
			}
			if l.current() == '}' {
				value += string(l.current())
				l.advance()
			}
		} else if l.current() == '$' && l.peek() == '(' {
			// Handle $(variable) interpolation within paths
			// Read $(
			value += string(l.current()) // $
			l.advance()
			value += string(l.current()) // (
			l.advance()

			// Read until closing )
			for l.current() != 0 && l.current() != ')' {
				value += string(l.current())
				l.advance()
			}
			if l.current() == ')' {
				value += string(l.current()) // )
				l.advance()
			}
		} else {
			value += string(l.current())
			l.advance()
		}

		// Stop if we hit a space followed by a brace (likely end of path before match block)
		if l.isWhitespace(l.current()) {
			next := l.peek()
			if next == '{' {
				break
			}
		}
	}

	return l.makeToken(PATH, value, startLine, startColumn), nil
}

func (l *Lexer) readIdentifier() (Token, error) {
	startLine := l.line
	startColumn := l.column
	value := ""

	for l.isAlphaNumeric(l.current()) || l.current() == '.' {
		value += string(l.current())
		l.advance()
	}

	// Check if it's a keyword
	tokenType := l.getKeywordType(value)
	return l.makeToken(tokenType, value, startLine, startColumn), nil
}

func (l *Lexer) readNumber() (Token, error) {
	startLine := l.line
	startColumn := l.column
	value := ""

	for l.isDigit(l.current()) {
		value += string(l.current())
		l.advance()
	}

	return l.makeToken(NUMBER, value, startLine, startColumn), nil
}

func (l *Lexer) getKeywordType(value string) TokenType {
	switch value {
	case "rules_version":
		return RULES_VERSION
	case "service":
		return SERVICE
	case "match":
		return MATCH
	case "allow":
		return ALLOW
	case "deny":
		return DENY
	case "if":
		return IF
	default:
		return IDENTIFIER
	}
}

// Métodos auxiliares del parser
func (p *ModernParser) check(tokenType TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == tokenType
}

func (p *ModernParser) peek() Token {
	return p.tokens[p.current]
}

func (p *ModernParser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *ModernParser) previous() Token {
	return p.tokens[p.current-1]
}

func (p *ModernParser) isAtEnd() bool {
	return p.peek().Type == EOF
}

func (p *ModernParser) consume(tokenType TokenType) bool {
	if p.check(tokenType) {
		p.advance()
		return true
	}
	return false
}

func (p *ModernParser) error(message string) error {
	token := p.peek()
	return fmt.Errorf("%s at line %d, column %d", message, token.Line, token.Column)
}

func (p *ModernParser) parseRulesVersion() (string, error) {
	if !p.consume(RULES_VERSION) {
		return "", p.error("expected 'rules_version'")
	}

	if !p.consume(EQUALS) {
		return "", p.error("expected '=' after 'rules_version'")
	}

	if !p.check(STRING) {
		return "", p.error("expected version string")
	}
	version := p.advance().Value

	if !p.consume(SEMICOLON) {
		return "", p.error("expected ';' after version")
	}

	return version, nil
}

func (p *ModernParser) parseMatchBlock() (*domain.MatchBlock, error) {
	if !p.consume(MATCH) {
		return nil, p.error("expected 'match'")
	}
	// Parse path - puede ser PATH, STRING o sequence of tokens forming a path
	var path string
	if p.check(PATH) {
		path = p.advance().Value
	} else if p.check(STRING) {
		path = p.advance().Value
	} else {
		// Handle composite paths like /users/{userId}/posts/{postId}
		path = ""
		for !p.check(LBRACE) && !p.isAtEnd() {
			token := p.peek()
			// Build path from sequence of tokens
			if token.Type == IDENTIFIER || token.Type == LBRACKET || token.Type == RBRACKET ||
				token.Type == DOT || token.Type == DOLLAR || token.Type == EQUALS ||
				(token.Value == "/" || strings.Contains(token.Value, "/")) {
				if path != "" && !strings.HasSuffix(path, "/") && !strings.HasPrefix(token.Value, "/") && token.Value != "/" {
					path += "" // Don't add space for path components
				}
				path += p.advance().Value
			} else {
				break
			}
		}
		if path == "" {
			return nil, p.error("expected path after 'match'")
		}
	}

	if !p.consume(LBRACE) {
		return nil, p.error("expected '{' after match path")
	}

	// Create match block
	matchBlock := &domain.MatchBlock{
		Path:      path,
		Variables: make(map[string]string),
		Allow:     make([]*domain.AllowStatement, 0),
		Deny:      make([]*domain.DenyStatement, 0),
		Nested:    make([]*domain.MatchBlock, 0),
	}

	// Parse variables from path
	p.extractVariables(matchBlock)

	// Parse content
	for !p.check(RBRACE) && !p.isAtEnd() {
		if p.check(ALLOW) {
			allowStmt, err := p.parseAllowStatement()
			if err != nil {
				return nil, err
			}
			matchBlock.Allow = append(matchBlock.Allow, allowStmt)
		} else if p.check(DENY) {
			denyStmt, err := p.parseDenyStatement()
			if err != nil {
				return nil, err
			}
			matchBlock.Deny = append(matchBlock.Deny, denyStmt)
		} else if p.check(MATCH) {
			nestedMatch, err := p.parseMatchBlock()
			if err != nil {
				return nil, err
			}
			matchBlock.Nested = append(matchBlock.Nested, nestedMatch)
		} else {
			// Skip unknown tokens with better error handling
			token := p.peek()
			if token.Type != EOF {
				return nil, p.error(fmt.Sprintf("unexpected token '%s' in match block", token.Value))
			}
			break
		}
	}
	if !p.consume(RBRACE) {
		return nil, p.error("unclosed match block - expected '}'")
	}

	return matchBlock, nil
}

func (p *ModernParser) parseAllowStatement() (*domain.AllowStatement, error) {
	if !p.consume(ALLOW) {
		return nil, p.error("expected 'allow'")
	}

	// Parse operations
	operations := make([]string, 0)
	if !p.check(COLON) {
		// First operation
		if !p.check(IDENTIFIER) {
			return nil, p.error("expected operation after 'allow'")
		}
		operations = append(operations, p.advance().Value)

		// Additional operations
		for p.consume(COMMA) {
			if !p.check(IDENTIFIER) {
				return nil, p.error("expected operation after ','")
			}
			operations = append(operations, p.advance().Value)
		}
	}

	if !p.consume(COLON) {
		return nil, p.error("expected ':' after operations")
	}

	if !p.consume(IF) {
		return nil, p.error("expected 'if' after ':'")
	} // Parse condition
	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}

	// Smart semicolon handling: be lenient when safe, strict when ambiguous
	nextToken := p.peek()
	if !p.consume(SEMICOLON) {
		// If next token is clearly end of block or another statement, be lenient
		if nextToken.Type == RBRACE || nextToken.Type == ALLOW || nextToken.Type == DENY || nextToken.Type == EOF {
			// Safe to omit semicolon - no ambiguity
		} else {
			// Ambiguous case - require semicolon for clarity
			return nil, p.error("expected ';' after allow statement condition")
		}
	}

	return &domain.AllowStatement{
		Operations: operations,
		Condition:  condition,
		Line:       p.current,
	}, nil
}

func (p *ModernParser) parseDenyStatement() (*domain.DenyStatement, error) {
	if !p.consume(DENY) {
		return nil, p.error("expected 'deny'")
	}

	// Parse operations (similar to allow)
	operations := make([]string, 0)
	if !p.check(COLON) {
		if !p.check(IDENTIFIER) {
			return nil, p.error("expected operation after 'deny'")
		}
		operations = append(operations, p.advance().Value)

		for p.consume(COMMA) {
			if !p.check(IDENTIFIER) {
				return nil, p.error("expected operation after ','")
			}
			operations = append(operations, p.advance().Value)
		}
	}
	if !p.consume(COLON) {
		return nil, p.error("expected ':' after operations")
	}

	if !p.consume(IF) {
		return nil, p.error("expected 'if' after ':'")
	}
	condition, err := p.parseCondition()
	if err != nil {
		return nil, err
	}

	// Smart semicolon handling: be lenient when safe, strict when ambiguous
	nextToken := p.peek()
	if !p.consume(SEMICOLON) {
		// If next token is clearly end of block or another statement, be lenient
		if nextToken.Type == RBRACE || nextToken.Type == ALLOW || nextToken.Type == DENY || nextToken.Type == EOF {
			// Safe to omit semicolon - no ambiguity
		} else {
			// Ambiguous case - require semicolon for clarity
			return nil, p.error("expected ';' after deny statement condition")
		}
	}

	return &domain.DenyStatement{
		Operations: operations,
		Condition:  condition,
		Line:       p.current,
	}, nil
}

func (p *ModernParser) parseCondition() (string, error) {
	// Parse complex conditions with proper tokenization
	condition := ""
	parenCount := 0
	braceCount := 0
	bracketCount := 0
	startPosition := p.current

	// Find the end of condition - it could be marked by:
	// 1. A semicolon at the top level (end of allow/deny statement)
	// 2. A closing brace that ends the match block
	// 3. The start of a new allow/deny statement
	for !p.isAtEnd() {
		token := p.peek()

		// Stop at semicolon if not inside any brackets/braces/parentheses
		if token.Type == SEMICOLON && parenCount == 0 && braceCount == 0 && bracketCount == 0 {
			break
		}

		// Stop at closing brace if we're at the top level (end of match block)
		if token.Type == RBRACE && parenCount == 0 && braceCount == 0 && bracketCount == 0 {
			break
		}

		// Stop if we encounter allow or deny at top level (start of new statement)
		if (token.Type == ALLOW || token.Type == DENY) && parenCount == 0 && braceCount == 0 && bracketCount == 0 {
			break
		} // Track parentheses for multiline conditions
		if token.Type == LPAREN {
			parenCount++
		} else if token.Type == RPAREN {
			parenCount--
			// Check for negative count (more closing than opening)
			if parenCount < 0 {
				// Reset counter but continue - this might be the end of our condition
				parenCount = 0
				break
			}
		} else if token.Type == LBRACE {
			braceCount++
		} else if token.Type == RBRACE {
			// Check if this is a match block closing brace
			if braceCount <= 0 && parenCount == 0 && bracketCount == 0 {
				break // Stop at match block closing brace
			}
			braceCount--
			// Check for negative count
			if braceCount < 0 {
				braceCount = 0
			}
		} else if token.Type == LBRACKET {
			bracketCount++
		} else if token.Type == RBRACKET {
			bracketCount--
			// Check for negative count
			if bracketCount < 0 {
				bracketCount = 0
			}
		}

		// Add space between tokens for readability, except for dots and special cases
		if condition != "" &&
			!shouldOmitSpace(p.tokens[p.current-1].Type, token.Type) {
			condition += " "
		}

		condition += p.advance().Value
	}
	if condition == "" {
		// Provide better error context
		context := ""
		if startPosition > 0 && startPosition < len(p.tokens) {
			context = fmt.Sprintf(" (started at token '%s')", p.tokens[startPosition].Value)
		}
		return "", p.error(fmt.Sprintf("empty condition%s", context))
	}

	// Be more lenient with validation - don't validate balanced parentheses
	// since they might be from outer scopes and the lexer handles syntax correctly
	return strings.TrimSpace(condition), nil
}

// shouldOmitSpace determines if space should be omitted between two token types
func shouldOmitSpace(prevType, currType TokenType) bool {
	// No space after dots
	if prevType == DOT {
		return true
	}
	// No space before dots
	if currType == DOT {
		return true
	}
	// No space after opening brackets/parens
	if prevType == LBRACKET || prevType == LPAREN {
		return true
	}
	// No space before closing brackets/parens
	if currType == RBRACKET || currType == RPAREN {
		return true
	}
	// No space around commas in function calls
	if prevType == COMMA || currType == COMMA {
		return true
	}
	// No space between function name and opening parenthesis
	if prevType == IDENTIFIER && currType == LPAREN {
		return true
	}
	return false
}

func (p *ModernParser) extractVariables(block *domain.MatchBlock) {
	path := block.Path

	// Extract variables like {userId}, {document=**}
	start := 0
	for {
		openBrace := strings.Index(path[start:], "{")
		if openBrace == -1 {
			break
		}
		openBrace += start

		closeBrace := strings.Index(path[openBrace:], "}")
		if closeBrace == -1 {
			break
		}
		closeBrace += openBrace

		variable := path[openBrace+1 : closeBrace]

		// Handle wildcard variables like document=**
		if equalIndex := strings.Index(variable, "="); equalIndex != -1 {
			varName := variable[:equalIndex]
			varValue := "{" + variable + "}"
			block.Variables[varName] = varValue
		} else {
			block.Variables[variable] = "{" + variable + "}"
		}

		start = closeBrace + 1
	}
}

func (p *ModernParser) validateSemantics(ruleset *domain.FirestoreRuleset) error {
	// Validaciones semánticas
	if ruleset.Service == "" {
		p.errors = append(p.errors, domain.ParseError{
			Line:    1,
			Message: "Missing service declaration",
			Type:    "semantic",
		})
	}

	// Validar que haya al menos un match block
	if len(ruleset.Matches) == 0 {
		p.errors = append(p.errors, domain.ParseError{
			Line:    1,
			Message: "No match blocks found",
			Type:    "semantic",
		})
	}

	// Validar paths y variables
	for _, match := range ruleset.Matches {
		if err := p.validateMatchBlock(match); err != nil {
			return err
		}
	}

	return nil
}

func (p *ModernParser) validateMatchBlock(block *domain.MatchBlock) error {
	// Validar que el path sea válido
	if block.Path == "" {
		p.errors = append(p.errors, domain.ParseError{
			Line:    1,
			Message: "Empty match path",
			Type:    "semantic",
		})
	}

	// Validar nested blocks
	for _, nested := range block.Nested {
		if err := p.validateMatchBlock(nested); err != nil {
			return err
		}
	}

	return nil
}

// Validate implementa validación sin parsing completo
func (p *ModernParser) Validate(ctx context.Context, content io.Reader) ([]domain.ParseError, error) {
	result, err := p.Parse(ctx, content)
	if err != nil {
		return nil, err
	}
	return result.Errors, nil
}

// GetMetrics retorna métricas del parser
func (p *ModernParser) GetMetrics() *domain.ParserMetrics {
	var averageParseTime time.Duration
	if p.totalParsed > 0 {
		averageParseTime = time.Duration(int64(p.totalParseTime) / p.totalParsed)
	}

	var errorRate float64
	if p.totalParsed > 0 {
		errorRate = float64(p.parseErrors) / float64(p.totalParsed)
	}

	return &domain.ParserMetrics{
		TotalParsed:      p.totalParsed,
		LastParseTime:    p.lastParseTime,
		AverageParseTime: averageParseTime,
		ErrorRate:        errorRate,
		CacheHitRate:     0.0,  // Not implemented yet
		MemoryUsage:      1024, // Placeholder for now
	}
}

// countRules cuenta recursivamente todas las reglas allow/deny en el ruleset
func (p *ModernParser) countRules(ruleset *domain.FirestoreRuleset) int {
	if ruleset == nil {
		return 0
	}

	count := 0
	for _, match := range ruleset.Matches {
		count += p.countRulesInMatch(match)
	}
	return count
}

// countRulesInMatch cuenta recursivamente las reglas en un match block
func (p *ModernParser) countRulesInMatch(match *domain.MatchBlock) int {
	if match == nil {
		return 0
	}

	count := 0

	// Contar reglas allow
	count += len(match.Allow)

	// Contar reglas deny
	count += len(match.Deny)

	// Contar reglas en matches anidados
	for _, nestedMatch := range match.Nested {
		count += p.countRulesInMatch(nestedMatch)
	}

	return count
}

// countLines cuenta el número de líneas en el contenido
func (p *ModernParser) countLines(content string) int {
	if content == "" {
		return 0
	}
	lines := strings.Split(content, "\n")
	return len(lines)
}

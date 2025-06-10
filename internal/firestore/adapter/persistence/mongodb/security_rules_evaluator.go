package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"firestore-clone/internal/firestore/domain/repository"

	"go.uber.org/zap"
)

// EvaluateAccess checks if a user can perform an operation on a resource
func (e *SecurityRulesEngine) EvaluateAccess(ctx context.Context, operation repository.OperationType, securityContext *repository.SecurityContext) (*repository.RuleEvaluationResult, error) {
	// Validate input parameters
	if ctx == nil {
		return &repository.RuleEvaluationResult{
			Allowed: false,
			Reason:  "Invalid context - nil context provided",
		}, fmt.Errorf("context cannot be nil")
	}

	if securityContext == nil {
		return &repository.RuleEvaluationResult{
			Allowed: false,
			Reason:  "Invalid security context - nil security context provided",
		}, fmt.Errorf("security context cannot be nil")
	}

	if securityContext.ProjectID == "" || securityContext.DatabaseID == "" {
		return &repository.RuleEvaluationResult{
			Allowed: false,
			Reason:  "Invalid security context - missing project ID or database ID",
		}, fmt.Errorf("project ID and database ID are required")
	}

	// Load rules for this project/database
	cacheKey := fmt.Sprintf("%s:%s", securityContext.ProjectID, securityContext.DatabaseID)
	rules, err := e.getCachedRules(ctx, cacheKey, securityContext.ProjectID, securityContext.DatabaseID)
	if err != nil {
		return nil, err
	}

	// If no rules are defined, default to deny
	if len(rules) == 0 {
		return &repository.RuleEvaluationResult{
			Allowed: false,
			Reason:  "No security rules defined - default deny",
		}, nil
	}

	// Evaluate rules in priority order
	for _, rule := range rules {
		if e.pathMatches(rule.Match, securityContext.Path) {
			// Check deny conditions first
			if denyCondition, hasDeny := rule.Deny[operation]; hasDeny {
				if denied, err := e.evaluateCondition(denyCondition, securityContext); err != nil {
					e.log.Error("Error evaluating deny condition",
						zap.String("condition", denyCondition),
						zap.Error(err))
					continue
				} else if denied {
					return &repository.RuleEvaluationResult{
						Allowed:  false,
						DeniedBy: rule.Match,
						Reason:   fmt.Sprintf("Denied by rule: %s", denyCondition),
					}, nil
				}
			}

			// Check allow conditions
			if allowCondition, hasAllow := rule.Allow[operation]; hasAllow {
				if allowed, err := e.evaluateCondition(allowCondition, securityContext); err != nil {
					e.log.Error("Error evaluating allow condition",
						zap.String("condition", allowCondition),
						zap.Error(err))
					continue
				} else if allowed {
					return &repository.RuleEvaluationResult{
						Allowed:   true,
						AllowedBy: rule.Match,
						Reason:    fmt.Sprintf("Allowed by rule: %s", allowCondition),
					}, nil
				}
			}
		}
	}

	// No matching rules found - default deny
	return &repository.RuleEvaluationResult{
		Allowed: false,
		Reason:  "No matching allow rules found - default deny",
	}, nil
}

// pathMatches checks if a path matches a pattern
func (e *SecurityRulesEngine) pathMatches(pattern, path string) bool {
	// Convert Firestore-style pattern to regex
	// Replace {variable} with regex capture groups
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\{[^}]+\\}", "[^/]+")
	regexPattern = strings.ReplaceAll(regexPattern, "\\*\\*", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", "[^/]*")
	regexPattern = "^" + regexPattern + "$"

	matched, _ := regexp.MatchString(regexPattern, path)
	return matched
}

// evaluateCondition evalúa una condición de regla de seguridad Firestore (CEL-like)
func (e *SecurityRulesEngine) evaluateCondition(condition string, securityContext *repository.SecurityContext) (bool, error) {
	// El parser es simple pero soporta expresiones lógicas, comparaciones, acceso a campos y funciones básicas
	parsed, err := parseCELExpr(condition)
	if err != nil {
		e.log.Warn("Error parsing CEL expression", zap.String("condition", condition), zap.Error(err))
		return false, err
	}
	result, err := evalCELExpr(parsed, securityContext)
	if err != nil {
		e.log.Warn("Error evaluating CEL expression", zap.String("condition", condition), zap.Error(err))
		return false, err
	}
	b, ok := result.(bool)
	if !ok {
		return false, nil
	}
	return b, nil
}

// --- CEL-like Expression Parser & Evaluator ---
// (Esto es un parser/evaluador recursivo simple, no un parser completo de CEL, pero cubre la mayoría de reglas Firestore)

// celExpr representa un nodo de expresión
// Puede ser extendido para más tipos y funciones

type celExpr interface{}

type celBinary struct {
	Op    string
	Left  celExpr
	Right celExpr
}
type celUnary struct {
	Op   string
	Expr celExpr
}
type celIdent struct {
	Name string
}
type celLiteral struct {
	Value interface{}
}
type celCall struct {
	Func string
	Args []celExpr
}

// parseCELExpr convierte una string en un árbol de expresiones (muy simplificado)
func parseCELExpr(expr string) (celExpr, error) {
	tokens := tokenizeCEL(expr)
	exprNode, rest, err := parseCELOr(tokens)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("unexpected tokens: %v", rest)
	}
	return exprNode, nil
}

// tokenizeCEL separa la expresión en tokens básicos
func tokenizeCEL(expr string) []string {
	re := regexp.MustCompile(`\s+`)
	expr = re.ReplaceAllString(expr, " ")
	tokens := []string{}
	token := ""
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if c == '(' || c == ')' || c == '!' || c == ',' {
			if token != "" {
				tokens = append(tokens, token)
				token = ""
			}
			tokens = append(tokens, string(c))
			continue
		}
		if c == ' ' {
			if token != "" {
				tokens = append(tokens, token)
				token = ""
			}
			continue
		}
		token += string(c)
	}
	if token != "" {
		tokens = append(tokens, token)
	}
	return tokens
}

// parseCELTokens: parser recursivo para expresiones lógicas y comparaciones
func parseCELTokens(tokens []string) (celExpr, error) {
	expr, rest, err := parseCELOr(tokens)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("unexpected tokens: %v", rest)
	}
	return expr, nil
}

func parseCELOr(tokens []string) (celExpr, []string, error) {
	expr, rest, err := parseCELAnd(tokens)
	if err != nil {
		return nil, nil, err
	}
	for len(rest) > 1 && rest[0] == "||" {
		right, rrest, err := parseCELAnd(rest[1:])
		if err != nil {
			return nil, nil, err
		}
		expr = &celBinary{Op: "||", Left: expr, Right: right}
		rest = rrest
	}
	return expr, rest, nil
}

func parseCELAnd(tokens []string) (celExpr, []string, error) {
	expr, rest, err := parseCELUnary(tokens)
	if err != nil {
		return nil, nil, err
	}
	for len(rest) > 1 && rest[0] == "&&" {
		right, rrest, err := parseCELUnary(rest[1:])
		if err != nil {
			return nil, nil, err
		}
		expr = &celBinary{Op: "&&", Left: expr, Right: right}
		rest = rrest
	}
	return expr, rest, nil
}

func parseCELUnary(tokens []string) (celExpr, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("empty expression")
	}
	if tokens[0] == "!" {
		expr, rest, err := parseCELUnary(tokens[1:])
		if err != nil {
			return nil, nil, err
		}
		return &celUnary{Op: "!", Expr: expr}, rest, nil
	}
	return parseCELPrimary(tokens)
}

func parseCELPrimary(tokens []string) (celExpr, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("empty primary")
	}
	if tokens[0] == "(" {
		expr, rest, err := parseCELOr(tokens[1:])
		if err != nil {
			return nil, nil, err
		}
		if len(rest) == 0 || rest[0] != ")" {
			return nil, nil, fmt.Errorf("missing closing parenthesis")
		}
		return expr, rest[1:], nil
	}
	// Función: in, size, etc.
	if len(tokens) > 2 && tokens[1] == "(" {
		funcName := tokens[0]
		args := []celExpr{}
		rest := tokens[2:]
		for len(rest) > 0 && rest[0] != ")" {
			arg, rrest, err := parseCELOr(rest)
			if err != nil {
				return nil, nil, err
			}
			args = append(args, arg)
			rest = rrest
			if len(rest) > 0 && rest[0] == "," {
				rest = rest[1:]
			}
		}
		if len(rest) == 0 || rest[0] != ")" {
			return nil, nil, fmt.Errorf("missing closing parenthesis in call")
		}
		return &celCall{Func: funcName, Args: args}, rest[1:], nil
	}
	// Comparaciones
	if len(tokens) > 2 && (tokens[1] == "==" || tokens[1] == "!=" || tokens[1] == "<" || tokens[1] == ">" || tokens[1] == "<=" || tokens[1] == ">=") {
		left := &celIdent{Name: tokens[0]}
		op := tokens[1]
		right, _, err := parseCELPrimary(tokens[2:])
		if err != nil {
			return nil, nil, err
		}
		return &celBinary{Op: op, Left: left, Right: right}, tokens[len(tokens):], nil
	}
	// Literal bool
	if tokens[0] == "true" {
		return &celLiteral{Value: true}, tokens[1:], nil
	}
	if tokens[0] == "false" {
		return &celLiteral{Value: false}, tokens[1:], nil
	}
	if tokens[0] == "null" {
		return &celLiteral{Value: nil}, tokens[1:], nil
	}
	// Número
	if n, err := parseCELNumber(tokens[0]); err == nil {
		return &celLiteral{Value: n}, tokens[1:], nil
	}
	// String literal ("...")
	if strings.HasPrefix(tokens[0], "\"") && strings.HasSuffix(tokens[0], "\"") {
		return &celLiteral{Value: tokens[0][1 : len(tokens[0])-1]}, tokens[1:], nil
	}
	// Identificador
	return &celIdent{Name: tokens[0]}, tokens[1:], nil
}

// parseCELNumber convierte string a float64
func parseCELNumber(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// evalCELExpr evalúa el árbol de expresión con el contexto de seguridad
func evalCELExpr(expr celExpr, ctx *repository.SecurityContext) (interface{}, error) {
	switch v := expr.(type) {
	case *celLiteral:
		return v.Value, nil
	case *celIdent:
		return resolveCELIdent(v.Name, ctx), nil
	case *celUnary:
		val, err := evalCELExpr(v.Expr, ctx)
		if err != nil {
			return nil, err
		}
		if v.Op == "!" {
			b, ok := val.(bool)
			return !b && ok, nil
		}
		return nil, fmt.Errorf("unsupported unary op: %s", v.Op)
	case *celBinary:
		left, err := evalCELExpr(v.Left, ctx)
		if err != nil {
			return nil, err
		}
		right, err := evalCELExpr(v.Right, ctx)
		if err != nil {
			return nil, err
		}
		switch v.Op {
		case "&&":
			return left.(bool) && right.(bool), nil
		case "||":
			return left.(bool) || right.(bool), nil
		case "==":
			return celEquals(left, right), nil
		case "!=":
			return !celEquals(left, right), nil
		case ">":
			return celCompare(left, right) > 0, nil
		case "<":
			return celCompare(left, right) < 0, nil
		case ">=":
			return celCompare(left, right) >= 0, nil
		case "<=":
			return celCompare(left, right) <= 0, nil
		default:
			return nil, fmt.Errorf("unsupported binary op: %s", v.Op)
		}
	case *celCall:
		args := []interface{}{}
		for _, a := range v.Args {
			val, err := evalCELExpr(a, ctx)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}
		return evalCELFunc(v.Func, args)
	default:
		return nil, fmt.Errorf("unknown expr type")
	}
}

// resolveCELIdent resuelve variables como auth.id, request, resource, timestamp
func resolveCELIdent(name string, ctx *repository.SecurityContext) interface{} {
	switch name {
	case "auth":
		return ctx.User
	case "auth.id":
		if ctx.User != nil {
			return ctx.User.ID
		}
		return nil
	case "request":
		return ctx.Request
	case "resource":
		return ctx.Resource
	case "timestamp":
		return ctx.Timestamp
	default:
		// Permite acceso a campos anidados tipo request.field
		if strings.HasPrefix(name, "request.") {
			k := strings.TrimPrefix(name, "request.")
			if v, ok := ctx.Request[k]; ok {
				return v
			}
		}
		if strings.HasPrefix(name, "resource.") {
			k := strings.TrimPrefix(name, "resource.")
			if v, ok := ctx.Resource[k]; ok {
				return v
			}
		}
		if strings.HasPrefix(name, "auth.") {
			k := strings.TrimPrefix(name, "auth.")
			if ctx.User != nil {
				switch k {
				case "email":
					return ctx.User.Email
				case "firstName":
					return ctx.User.FirstName
				case "lastName":
					return ctx.User.LastName
				}
			}
		}
		return nil
	}
}

// celEquals compara dos valores
func celEquals(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// celCompare compara dos valores numéricos o strings
func celCompare(a, b interface{}) int {
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		if af < bf {
			return -1
		} else if af > bf {
			return 1
		}
		return 0
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return strings.Compare(as, bs)
	}
	return 0
}

// evalCELFunc implementa funciones como in, size, etc.
func evalCELFunc(name string, args []interface{}) (interface{}, error) {
	switch name {
	case "in":
		if len(args) != 2 {
			return false, nil
		}
		needle := args[0]
		haystack, ok := args[1].([]interface{})
		if !ok {
			return false, nil
		}
		for _, v := range haystack {
			if celEquals(needle, v) {
				return true, nil
			}
		}
		return false, nil
	case "size":
		if len(args) != 1 {
			return 0, nil
		}
		switch v := args[0].(type) {
		case string:
			return float64(len(v)), nil
		case []interface{}:
			return float64(len(v)), nil
		}
		return 0, nil
	default:
		return nil, fmt.Errorf("unsupported function: %s", name)
	}
}

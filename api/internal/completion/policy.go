package completion

var validIntents = map[string]bool{
	"": true, "component-props": true, "member-access": true, "function-arguments": true,
	"tailwind-class": true, "object-fields": true, "import": true, "function-body": true,
	"condition-expression": true, "type-definition": true, "general": true,
}

func ValidIntent(intent string) bool { return validIntents[intent] }

func IntentInstruction(intent string) string {
	switch intent {
	case "component-props":
		return "Complete valid JSX/TSX props using the component declarations and existing local style."
	case "member-access":
		return "Complete only the most likely valid member access or its immediate expression."
	case "function-arguments":
		return "Complete arguments that match the referenced function signature and available variables."
	case "tailwind-class":
		return "Complete only relevant utility class names consistent with nearby classes."
	case "object-fields":
		return "Complete fields that match the expected type and do not duplicate existing fields."
	case "import":
		return "Complete one valid import using dependencies and symbols already present in the project context."
	case "function-body":
		return "Complete the smallest useful function body consistent with surrounding code and return type."
	case "condition-expression":
		return "Complete only the condition or expression using variables available in scope."
	case "type-definition":
		return "Complete a concise type definition consistent with its usages and project conventions."
	default:
		return "Prefer the smallest useful continuation consistent with the surrounding code."
	}
}

func MaxTokensForIntent(intent string, configured int) int {
	limit := configured
	switch intent {
	case "member-access":
		limit = 48
	case "function-arguments", "tailwind-class", "import", "condition-expression":
		limit = 64
	case "component-props":
		limit = 96
	case "object-fields":
		limit = 128
	case "type-definition":
		limit = 160
	}
	if configured < limit {
		return configured
	}
	return limit
}

func MaxLinesForIntent(intent string, configured int) int {
	switch intent {
	case "member-access", "function-arguments", "tailwind-class", "import", "condition-expression":
		return minInt(configured, 1)
	case "component-props":
		return minInt(configured, 2)
	case "object-fields":
		return minInt(configured, 8)
	case "type-definition":
		return minInt(configured, 12)
	default:
		return configured
	}
}

func SingleLineIntent(intent string) bool { return MaxLinesForIntent(intent, 2) == 1 }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

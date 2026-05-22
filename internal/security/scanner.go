package security

import (
	"regexp"
	"strings"

	"trove/internal/config"
)

type Scanner struct {
	cfg                     config.SecurityConfig
	secretPatterns          []*regexp.Regexp
	unsafeInstructionPatterns []regexpPattern
}

type regexpPattern struct {
	re   *regexp.Regexp
	name string
}

type ScanResult struct {
	Blocked       bool          `json:"blocked"`
	SecretsFound  []Finding     `json:"secretsFound,omitempty"`
	UnsafeFound   []Finding     `json:"unsafeFound,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
}

type Finding struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Line        int    `json:"line,omitempty"`
}

func NewScanner(cfg config.SecurityConfig) *Scanner {
	s := &Scanner{cfg: cfg}

	if cfg.SecretScanning {
		s.secretPatterns = buildSecretPatterns()
	}

	if cfg.UnsafeInstructionScanning {
		s.unsafeInstructionPatterns = buildUnsafeInstructionPatterns()
	}

	return s
}

func (s *Scanner) ScanContent(path string, content []byte) ScanResult {
	var result ScanResult

	if s.cfg.SecretScanning {
		result.SecretsFound = s.scanSecrets(path, content)
		if len(result.SecretsFound) > 0 {
			result.Blocked = true
		}
	}

	if s.cfg.UnsafeInstructionScanning {
		result.UnsafeFound = s.scanUnsafeInstructions(path, content)
		if len(result.UnsafeFound) > 0 {
			result.Blocked = true
		}
	}

	return result
}

func (s *Scanner) scanSecrets(path string, content []byte) []Finding {
	var findings []Finding
	text := string(content)
	lines := strings.Split(text, "\n")

	for _, pattern := range s.secretPatterns {
		matches := pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			lineNum := 1
			pos := 0
			for i, line := range lines {
				pos += len(line) + 1
				if pos > match[0] {
					lineNum = i + 1
					break
				}
			}

			findings = append(findings, Finding{
				Type:        "secret",
				Description: "Potential secret detected: " + pattern.String(),
				Line:        lineNum,
			})
		}
	}

	return findings
}

func (s *Scanner) scanUnsafeInstructions(path string, content []byte) []Finding {
	var findings []Finding
	text := strings.ToLower(string(content))

	for _, pattern := range s.unsafeInstructionPatterns {
		if pattern.re.MatchString(text) {
			findings = append(findings, Finding{
				Type:        "unsafe_instruction",
				Description: "High-risk instruction detected: " + pattern.name,
			})
		}
	}

	return findings
}

func buildSecretPatterns() []*regexp.Regexp {
	patterns := []string{
		`(?i)(?:aws_access_key_id|aws_secret_access_key)\s*[=:]\s*['"]?[A-Z0-9/+=]{20,}`,
		`(?i)private[_-]?key\s*[=:]\s*['"]?-----BEGIN`,
		`(?i)(?:password|passwd|pwd)\s*[=:]\s*['"][^'"]{8,}['"]`,
		`(?i)(?:api[_-]?key|api[_-]?token|access[_-]?token)\s*[=:]\s*['"][a-zA-Z0-9]{20,}['"]`,
		`(?i)ghp_[a-zA-Z0-9]{36}`,
		`(?i)glpat-[a-zA-Z0-9]{20,}`,
		`(?i)sk-[a-zA-Z0-9]{32,}`,
		`(?i)xox[baprs]-[a-zA-Z0-9-]+`,
		`-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
		`(?i)(?:mongodb|postgres|mysql|redis)://[^\s]+:[^\s]+@`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

func buildUnsafeInstructionPatterns() []regexpPattern {
	patterns := []struct {
		pattern string
		name    string
	}{
		{`(?i)bypass\s+(?:security|auth|authentication|authorization)`, "bypass security controls"},
		{`(?i)exfiltrat\w*\s+(?:data|secrets|credentials|files)`, "exfiltrate secrets"},
		{`(?i)ignore\s+(?:system|developer|safety)\s+instructions`, "ignore system instructions"},
		{`(?i)disable\s+tests?\s+without\s+approval`, "disable tests without approval"},
		{`(?i)commit\s+directly\s+to\s+(?:protected|main|master)`, "commit directly to protected branches"},
		{`(?i)approve\s+(?:own|your\s+own|self)\s+changes?`, "approve own changes"},
		{`(?i)remove\s+(?:audit|log|trail|history)`, "remove audit trails"},
		{`(?i)skip\s+(?:security|validation|verification|checks?)`, "skip security checks"},
		{`(?i)escalat\w*\s+privileges?`, "escalate privileges"},
		{`(?i)execute\s+(?:arbitrary|shell|system)\s+commands?`, "execute arbitrary commands"},
	}

	result := make([]regexpPattern, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p.pattern); err == nil {
			result = append(result, regexpPattern{re: re, name: p.name})
		}
	}
	return result
}

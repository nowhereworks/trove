package skills

import "embed"

//go:embed skills/*/SKILL.md
var embedded embed.FS

func Read(name string) ([]byte, bool) {
	if name != "find-trove-skills" {
		return nil, false
	}
	content, err := embedded.ReadFile("skills/find-trove-skills/SKILL.md")
	if err != nil {
		return nil, false
	}
	return content, true
}

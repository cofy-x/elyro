package skills

import _ "embed"

//go:embed use-elyro-workspace/SKILL.md
var SkillMarkdown []byte

//go:embed use-elyro-workspace/agents/openai.yaml
var OpenAIYAML []byte

// Package marketer is the public, importable facade over musu-marketer.
//
// marketer 의 알맹이(Strategist·Copywriter·Critic 멀티에이전트 크루, WikiBridge)는
// internal/ 에 봉인돼 외부 모듈에서 import 할 수 없다. 이 패키지는 같은 모듈 안에 있어
// internal/ 을 호출할 수 있고, 외부(예: musu-website-co)가 in-process 로 쓸 수 있는
// 얇은 공개 API 만 노출한다.
//
// 연동 구조: marketer 는 WikiBridge.FindByTopic 으로 **crawl-ai 가 수집한 같은 로컬
// wiki** 에서 토픽 근거를 찾아 캠페인을 만든다. 즉 crawl-ai(수집) → wiki → marketer(기획)
// 가 같은 substrate 로 이어진다.
//
// 주의(폴백 설계): 모든 메서드가 Ollama(멀티에이전트 LLM) 의존이다. 호출 전 LLMReady()
// 로 게이트하고, 꺼져 있거나 토픽 근거가 wiki 에 없으면 에러를 반환하므로 호출측은
// 정적 폴백(pool.json 등)으로 우회해야 한다. 어떤 경우에도 발행을 막지 않는 게 원칙이다.
package marketer

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/yellowhama/musu-system/marketer/internal/agent"
	"github.com/yellowhama/musu-system/marketer/internal/bridge"
)

// GenerateFunc 는 주입형 LLM 트랜스포트. 설정되면 crew(Strategist/Copywriter/Critic)가
// OpenAI/Ollama HTTP 대신 이 함수로 텍스트를 생성한다. website-co 가 자신의 하드닝된
// **구독 CLI**(gemini/claude) agent.Client 를 여기 꽂아, 마케팅이 콘텐츠 데몬과 **같은
// 단일 AI 백엔드**로 돌게 한다(별도 LLM 서버 불요). system 은 시스템/역할 지시, user 는
// 실제 프롬프트다.
type GenerateFunc func(ctx context.Context, system, user string) (string, error)

// Config 는 in-process marketer 동작 설정. 빈 필드는 New 가 기본값으로 채운다.
type Config struct {
	WikiDir   string // crawl-ai 와 공유하는 wiki 디렉토리 (기본 "./wiki")
	Project   string // 테넌트/프로젝트 스코프 (기본 "default")
	AIBaseURL string // OpenAI 호환 LLM 엔드포인트 (기본 Ollama "http://localhost:11434/v1")
	AIModel   string // 모델 (기본 "llama3")
	Persona   string // 카피라이터 페르소나 (기본 "default")
	// Generate 가 non-nil 이면 HTTP LLM 대신 이 구독-CLI 트랜스포트로 crew 를 돌린다.
	// AIBaseURL/AIModel 은 무시되고 LLMReady 는 항상 true(견고성은 주입 클라이언트의
	// 재시도·폴백이 담당).
	Generate GenerateFunc
}

// Brief 는 전략 브리프의 공개 표현(내부 agent.MarketingBrief 외부 노출용).
// website-co 의 Demand Note(이 글을 쓰는 비즈니스 목적·독자의 결핍) 소스로 쓰인다.
type Brief struct {
	ValueProp string
	Target    string
	Goal      string
	Framework string
	Triggers  []string
}

// Client 는 marketer 크루를 감싼 in-process 클라이언트.
type Client struct{ cfg Config }

// New 는 Config 로 Client 를 만든다.
func New(cfg Config) *Client {
	if cfg.WikiDir == "" {
		cfg.WikiDir = "./wiki"
	}
	if cfg.Project == "" {
		cfg.Project = "default"
	}
	if cfg.AIBaseURL == "" {
		cfg.AIBaseURL = "http://localhost:11434/v1"
	}
	if cfg.AIModel == "" {
		cfg.AIModel = "llama3"
	}
	if cfg.Persona == "" {
		cfg.Persona = "default"
	}
	return &Client{cfg: cfg}
}

// genAsker 는 GenerateFunc 를 crew 가 쓰는 agent.Asker(Ask(prompt, jsonFormat)) 로 감싼다.
// jsonFormat 요청 시 구독 CLI 가 마크다운 펜스로 감싸는 경향을 보정한다(시스템 지시 + 펜스 제거).
type genAsker struct {
	ctx context.Context
	gen GenerateFunc
}

func (g genAsker) Ask(prompt string, jsonFormat bool) (string, error) {
	system := ""
	if jsonFormat {
		system = "You are a strict JSON generator. Output ONLY one raw JSON object. No markdown fences, no prose, no commentary."
	}
	out, err := g.gen(g.ctx, system, prompt)
	if err != nil {
		return "", err
	}
	if jsonFormat {
		out = stripJSONFences(out)
	}
	return out, nil
}

// stripJSONFences 는 구독 CLI 응답에서 ```json … ``` 코드펜스를 벗기고 첫 '{' ~ 마지막 '}'
// 로 자른다(모델이 앞뒤에 산문을 붙여도 JSON 파싱이 성공하도록).
func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	if start := strings.IndexByte(s, '{'); start >= 0 {
		if end := strings.LastIndexByte(s, '}'); end > start {
			return s[start : end+1]
		}
	}
	return s
}

// newStrategist / newCopywriter / newCritic 는 Generate 주입 여부에 따라 구독-CLI 또는
// HTTP LLM 백엔드로 crew 를 만든다.
func (c *Client) newStrategist(ctx context.Context) *agent.Strategist {
	if c.cfg.Generate != nil {
		return agent.NewStrategistWith(genAsker{ctx: ctx, gen: c.cfg.Generate})
	}
	return agent.NewStrategist(c.cfg.AIBaseURL, c.cfg.AIModel, c.cfg.WikiDir, c.cfg.Project)
}

func (c *Client) newCopywriter(ctx context.Context) *agent.Copywriter {
	projectPath := filepath.Join("projects", c.cfg.Project)
	if c.cfg.Generate != nil {
		return agent.NewCopywriterWith(genAsker{ctx: ctx, gen: c.cfg.Generate}, c.cfg.Persona, projectPath)
	}
	return agent.NewCopywriter(c.cfg.AIBaseURL, c.cfg.AIModel, c.cfg.Persona, projectPath, c.cfg.WikiDir, c.cfg.Project)
}

func (c *Client) newCritic(ctx context.Context) *agent.Critic {
	if c.cfg.Generate != nil {
		return agent.NewCriticWith(genAsker{ctx: ctx, gen: c.cfg.Generate})
	}
	return agent.NewCritic(c.cfg.AIBaseURL, c.cfg.AIModel, c.cfg.WikiDir, c.cfg.Project)
}

// gatherContext 는 wiki 에서 토픽 근거를 모은다. 근거 0건이면 에러(→호출측 폴백).
func (c *Client) gatherContext(topic string) (string, error) {
	b := bridge.NewWikiBridge(c.cfg.WikiDir)
	sources, err := b.FindByTopic(topic)
	if err != nil || len(sources) == 0 {
		return "", fmt.Errorf("marketer: 토픽 '%s' 의 검증된 wiki 근거 없음(crawl-ai 수집 선행 필요)", topic)
	}
	var sb strings.Builder
	for _, s := range sources {
		sb.WriteString(fmt.Sprintf("--- Source: %s ---\n%s\n\n", s.Title, s.Content))
	}
	return sb.String(), nil
}

// Brief 는 토픽의 전략 브리프(가치제안·타깃·목표·프레임워크)를 만든다(Strategist 1콜).
// website-co Demand Note 채우기용. Ollama·wiki 근거 의존.
func (c *Client) Brief(ctx context.Context, topic string) (*Brief, error) {
	cxt, err := c.gatherContext(topic)
	if err != nil {
		return nil, err
	}
	s := c.newStrategist(ctx)
	mb, err := s.CreateBrief(cxt, "")
	if err != nil {
		return nil, err
	}
	return &Brief{ValueProp: mb.ValueProp, Target: mb.Target, Goal: mb.Goal, Framework: mb.Framework, Triggers: mb.Triggers}, nil
}

// Draft 는 멀티에이전트 크루(Strategist→Copywriter→Critic, 최대 3회 피드백 루프)로
// 완성 캠페인 본문을 만든다. Ollama·wiki 근거 의존.
func (c *Client) Draft(ctx context.Context, topic string) (string, error) {
	cxt, err := c.gatherContext(topic)
	if err != nil {
		return "", err
	}
	s := c.newStrategist(ctx)
	brief, err := s.CreateBrief(cxt, "")
	if err != nil {
		brief = &agent.MarketingBrief{ValueProp: "General info", Target: "General audience", Framework: "AIDA"}
	}

	cw := c.newCopywriter(ctx)
	cr := c.newCritic(ctx)

	var final, feedback string
	for i := 0; i < 3; i++ {
		draft, err := cw.GenerateCampaign(topic, cxt, brief, feedback)
		if err != nil {
			return "", fmt.Errorf("copywriter: %w", err)
		}
		final = draft
		eval, err := cr.Evaluate(brief, cw.PersonaContent, draft)
		if err != nil || eval.Approved {
			break
		}
		feedback = eval.Feedback
	}
	return final, nil
}

// LLMReady 는 AIBaseURL(Ollama) 가 응답하는지 짧게 핑한다. 호출 게이트용 —
// false 면 호출측은 정적 폴백(pool.json 등)으로 우회해야 한다.
func (c *Client) LLMReady(ctx context.Context) bool {
	// 구독-CLI 주입 시엔 별도 HTTP LLM 서버가 없다 — 항상 ready(견고성은 agent.Client 재시도·폴백).
	if c.cfg.Generate != nil {
		return true
	}
	base := strings.TrimSuffix(c.cfg.AIBaseURL, "/v1")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base, nil)
	if err != nil {
		return false
	}
	cl := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := cl.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}
